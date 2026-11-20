package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"relayradar/internal/probe"
)

type probeIngest struct {
	Slug      string    `json:"slug"`
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	LatencyMS int       `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
}

func (s *Server) handleProbeConfig(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	if r.Method == http.MethodGet {
		raw, err := os.ReadFile(s.cfg.ProbeConfigPath)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"config": "", "exists": false})
			return
		}
		var cfg probe.Config
		if json.Unmarshal(raw, &cfg) == nil {
			for i := range cfg.Stations {
				if cfg.Stations[i].APIKey != "" {
					cfg.Stations[i].APIKey = "***"
				}
			}
		}
		out, _ := json.MarshalIndent(cfg, "", "  ")
		writeJSON(w, http.StatusOK, map[string]any{"config": string(out), "exists": true})
		return
	}
	var body struct {
		Config string `json:"config"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var cfg probe.Config
	if err := json.Unmarshal([]byte(body.Config), &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid probes JSON: "+err.Error())
		return
	}
	if len(cfg.Stations) == 0 {
		writeError(w, http.StatusBadRequest, "no stations configured")
		return
	}
	for _, t := range cfg.Stations {
		if t.Slug == "" || t.Service == "" || t.BaseURL == "" {
			writeError(w, http.StatusBadRequest, "each target needs slug, service and baseUrl")
			return
		}
	}
	if err := os.WriteFile(s.cfg.ProbeConfigPath, []byte(body.Config), 0o600); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "targets": len(cfg.Stations)})
}


// so it does not need a database connection.
func (s *Server) handleProbeIngest(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ProbeIngestToken == "" {
		writeError(w, http.StatusUnauthorized, "probe ingest disabled")
		return
	}
	auth := r.Header.Get("Authorization")
	want := "Bearer " + s.cfg.ProbeIngestToken
	if len(auth) != len(want) || subtle.ConstantTimeCompare([]byte(auth), []byte(want)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid probe ingest token")
		return
	}
	var body probeIngest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Status != "ok" && body.Status != "slow" && body.Status != "down" {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	groupID, err := s.svc.Repo().GroupByStationService(r.Context(), body.Slug, body.Service)
	if err != nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	at := body.CheckedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if err := s.svc.Repo().InsertProbe(r.Context(), groupID, body.Status, body.LatencyMS, at); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	s.svc.Invalidate()
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}
