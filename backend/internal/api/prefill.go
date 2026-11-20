package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// handlePrefill inspects a station with the user-provided key and returns the
// fields the submission form can fill automatically.
func (s *Server) handlePrefill(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	raw := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		writeError(w, http.StatusBadRequest, "invalid baseUrl")
		return
	}
	root := strings.TrimSuffix(raw, "/v1")
	client := &http.Client{Timeout: 15 * time.Second}

	get := func(target string, dst any) bool {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			return false
		}
		req.Header.Set("Authorization", "Bearer "+in.APIKey)
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return false
		}
		return json.NewDecoder(resp.Body).Decode(dst) == nil
	}

	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	modelOK := get(root+"/v1/models", &models)
	ids := make([]string, 0, len(models.Data))
	for _, m := range models.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}

	var pricing struct {
		Data []struct {
			Name       string  `json:"model_name"`
			QuotaType  int     `json:"quota_type"`
			ModelPrice float64 `json:"model_price"`
		} `json:"data"`
	}
	pricingOK := get(root+"/api/pricing", &pricing)
	type price struct {
		Name     string  `json:"name"`
		PriceUSD float64 `json:"priceUsd"`
	}
	prices := make([]price, 0, len(pricing.Data))
	for _, p := range pricing.Data {
		if p.Name != "" && p.QuotaType == 1 && p.ModelPrice > 0 {
			prices = append(prices, price{Name: p.Name, PriceUSD: p.ModelPrice})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"domain":     u.Hostname(),
		"baseUrl":    root,
		"models":     ids,
		"modelsOk":   modelOK,
		"pricing":    prices,
		"pricingOk":  pricingOK,
		"inspectedAt": time.Now().UTC(),
	})
}
