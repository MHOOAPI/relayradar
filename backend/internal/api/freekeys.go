package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"relayradar/internal/audit"
	"relayradar/internal/store"
)

// maskKey keeps a short prefix/suffix so submitters can recognise their key
// without exposing the secret in the list.
func maskKey(k string) string {
	if len(k) <= 10 {
		return "****"
	}
	return k[:5] + "****" + k[len(k)-4:]
}

type freeKeyView struct {
	ID        int64  `json:"id"`
	Provider  string `json:"provider"`
	BaseURL   string `json:"baseUrl"`
	KeyMasked string `json:"keyMasked"`
	Models    string `json:"models"`
	Note      string `json:"note"`
	Submitter string `json:"submitter"`
	Status    string `json:"status"`
	Claims    int    `json:"claims"`
	CreatedAt string `json:"createdAt"`
}

func toFreeKeyView(k store.FreeKey) freeKeyView {
	return freeKeyView{
		ID: k.ID, Provider: k.Provider, BaseURL: k.BaseURL, KeyMasked: maskKey(k.Key),
		Models: k.Models, Note: k.Note, Submitter: k.Submitter, Status: k.Status,
		Claims: k.Claims, CreatedAt: k.CreatedAt.Format(time.RFC3339),
	}
}

// checkFreeKey probes GET {base}/models to decide whether a submitted key works.
func (s *Server) checkFreeKey(ctx context.Context, baseURL, key string) (string, string) {
	if _, err := audit.ValidateTarget(baseURL, s.cfg.AllowLocalTargets); err != nil {
		return "unknown", ""
	}
	u := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(u, "/v1") {
		u += "/v1"
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"/models", nil)
	if err != nil {
		return "unknown", ""
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "unknown", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "invalid", ""
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "unknown", ""
	}
	var body struct {
		Data   []struct{ ID string `json:"id"` } `json:"data"`
		Models []string                         `json:"models"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	ids := make([]string, 0, len(body.Data)+len(body.Models))
	for _, m := range body.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	ids = append(ids, body.Models...)
	if len(ids) > 6 {
		ids = ids[:6]
	}
	return "available", strings.Join(ids, ", ")
}

func (s *Server) handleFreeKeyCreate(w http.ResponseWriter, r *http.Request) {
	if !s.freeKeyRL.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many submissions, retry later")
		return
	}
	var body struct {
		BaseURL   string `json:"baseUrl"`
		APIKey    string `json:"apiKey"`
		Models    string `json:"models"`
		Note      string `json:"note"`
		Submitter string `json:"submitter"`
		Website   string `json:"website"` // honeypot, must stay empty
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Website != "" {
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
		return
	}
	base := strings.TrimSpace(body.BaseURL)
	key := strings.TrimSpace(body.APIKey)
	if base == "" || key == "" {
		writeError(w, http.StatusBadRequest, "baseUrl and apiKey are required")
		return
	}
	if len(key) < 8 || len(key) > 300 {
		writeError(w, http.StatusBadRequest, "apiKey length looks wrong")
		return
	}
	if _, err := audit.ValidateTarget(base, s.cfg.AllowLocalTargets); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	parsed, err := url.Parse(base)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid baseUrl")
		return
	}
	provider := parsed.Hostname()

	status, models := s.checkFreeKey(r.Context(), base, key)
	if body.Models != "" {
		models = strings.TrimSpace(body.Models)
	}
	now := time.Now().UTC()
	id, err := s.svc.Repo().CreateFreeKey(r.Context(), store.FreeKey{
		Provider: provider, BaseURL: base, Key: key, Models: models,
		Note: strings.TrimSpace(body.Note), Submitter: strings.TrimSpace(body.Submitter),
		Status: status, LastCheckedAt: &now,
	})
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": status})
}

func (s *Server) handleFreeKeyList(w http.ResponseWriter, r *http.Request) {
	includeInvalid := r.URL.Query().Get("include_invalid") == "1"
	rows, err := s.svc.Repo().ListFreeKeys(r.Context(), includeInvalid, r.URL.Query().Get("q"))
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]freeKeyView, 0, len(rows))
	for _, k := range rows {
		items = append(items, toFreeKeyView(k))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func (s *Server) handleFreeKeyClaim(w http.ResponseWriter, r *http.Request) {
	if !s.claimRL.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many claims, retry later")
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	k, err := s.svc.Repo().GetFreeKey(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "key not found")
		return
	}
	if k.Status == "invalid" {
		writeError(w, http.StatusGone, "this key has been marked invalid")
		return
	}
	_ = s.svc.Repo().IncrementFreeKeyClaims(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]any{
		"apiKey": k.Key, "baseUrl": k.BaseURL, "models": k.Models, "note": k.Note,
		"warning": "Shared keys can be revoked or rate-limited at any time; never use them for production or billing.",
	})
}

func (s *Server) handleAdminFreeKeys(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	rows, err := s.svc.Repo().ListFreeKeys(r.Context(), true, "")
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]freeKeyView, 0, len(rows))
	for _, k := range rows {
		items = append(items, toFreeKeyView(k))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminFreeKeyDelete(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.svc.Repo().DeleteFreeKey(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "key not found")
		return
	}
	s.logAdminAction(r, admin, "freekey.delete", fmt.Sprintf("free key #%d", id), "deleted")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
