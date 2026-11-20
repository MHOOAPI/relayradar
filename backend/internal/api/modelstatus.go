package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"relayradar/internal/store"
)

type modelProvider struct {
	Key       string
	Name      string
	StatusURL string
	StatusAPI string
	ProbeURL  string
}

type modelStatusItem struct {
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	Status      string              `json:"status"`
	Description string              `json:"description"`
	StatusURL   string              `json:"statusUrl"`
	CheckedVia  string              `json:"checkedVia"`
	CheckedAt   time.Time           `json:"checkedAt"`
	Components  []providerComponent `json:"components,omitempty"`
	Incidents   []providerIncident  `json:"incidents,omitempty"`
	History     []statusDay         `json:"history,omitempty"`
}

type providerComponent struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type providerIncident struct {
	Name       string             `json:"name"`
	Status     string             `json:"status"`
	Impact     string             `json:"impact"`
	URL        string             `json:"url"`
	Created    string             `json:"created"`
	StartedAt  string             `json:"startedAt"`
	ResolvedAt string             `json:"resolvedAt"`
	Components []string           `json:"components,omitempty"`
	Updates    []providerUpdate   `json:"updates,omitempty"`
}

type providerUpdate struct {
	Status    string `json:"status"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

var modelProviders = []modelProvider{
	{Key: "openai", Name: "OpenAI", StatusURL: "https://status.openai.com", StatusAPI: "https://status.openai.com/api/v2/status.json", ProbeURL: "https://api.openai.com/v1/models"},
	{Key: "anthropic", Name: "Anthropic", StatusURL: "https://status.anthropic.com", StatusAPI: "https://status.anthropic.com/api/v2/status.json", ProbeURL: "https://api.anthropic.com/v1/models"},
	{Key: "google", Name: "Google Gemini", StatusURL: "https://status.cloud.google.com", ProbeURL: "https://generativelanguage.googleapis.com/v1beta/models"},
	{Key: "deepseek", Name: "DeepSeek", StatusURL: "https://status.deepseek.com", ProbeURL: "https://api.deepseek.com/v1/models"},
	{Key: "moonshot", Name: "Moonshot", StatusURL: "https://status.moonshot.cn", ProbeURL: "https://api.moonshot.cn/v1/models"},
	{Key: "alibaba", Name: "Alibaba Qwen", StatusURL: "https://status.aliyun.com", ProbeURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/models"},
	{Key: "xai", Name: "xAI", StatusURL: "https://status.x.ai", StatusAPI: "https://status.x.ai/api/v2/status.json", ProbeURL: "https://api.x.ai/v1/models"},
}

var statuspageIndicator = map[string]string{
	"none": "operational", "minor": "degraded", "major": "degraded", "critical": "outage",
}

func statuspageComponent(status string) string {
	switch status {
	case "operational":
		return "operational"
	case "degraded_performance", "partial_outage", "under_maintenance":
		return "degraded"
	case "major_outage":
		return "outage"
	default:
		return "unknown"
	}
}

func fetchJSON(ctx context.Context, client *http.Client, url string, out any) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	return json.NewDecoder(resp.Body).Decode(out) == nil
}

func probeModelProvider(ctx context.Context, p modelProvider) modelStatusItem {
	item := modelStatusItem{Key: p.Key, Name: p.Name, StatusURL: p.StatusURL, CheckedAt: time.Now().UTC()}
	client := &http.Client{Timeout: 4 * time.Second}

	if p.StatusAPI != "" {
		if req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.StatusAPI, nil); err == nil {
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				var body struct {
					Status struct {
						Indicator   string `json:"indicator"`
						Description string `json:"description"`
					} `json:"status"`
				}
				if resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&body) == nil && body.Status.Indicator != "" {
					item.Status = statuspageIndicator[body.Status.Indicator]
					if item.Status == "" {
						item.Status = "unknown"
					}
					item.Description = body.Status.Description
					item.CheckedVia = "statuspage"
					base := strings.TrimSuffix(p.StatusAPI, "/status.json")
					var comps struct {
						Components []struct {
							Name   string `json:"name"`
							Status string `json:"status"`
						} `json:"components"`
					}
					if fetchJSON(ctx, client, base+"/components.json", &comps) {
						for i, c := range comps.Components {
							if i >= 6 {
								break
							}
							item.Components = append(item.Components, providerComponent{Name: c.Name, Status: statuspageComponent(c.Status)})
						}
					}
					var incs struct {
						Incidents []struct {
							Name      string `json:"name"`
							Status    string `json:"status"`
							Impact    string `json:"impact"`
							Shortlink string `json:"shortlink"`
							CreatedAt string `json:"created_at"`
							StartedAt string `json:"started_at"`
							ResolvedAt string `json:"resolved_at"`
							Components []struct {
								Name string `json:"name"`
							} `json:"components"`
							IncidentUpdates []struct {
								Status    string `json:"status"`
								Body      string `json:"body"`
								CreatedAt string `json:"created_at"`
							} `json:"incident_updates"`
						} `json:"incidents"`
					}
					if fetchJSON(ctx, client, base+"/incidents.json", &incs) {
						for i, inc := range incs.Incidents {
							if i >= 20 {
								break
							}
							item2 := providerIncident{
								Name: inc.Name, Status: inc.Status, Impact: inc.Impact, URL: inc.Shortlink,
								Created: inc.CreatedAt, StartedAt: inc.StartedAt, ResolvedAt: inc.ResolvedAt,
							}
							for _, c := range inc.Components {
								item2.Components = append(item2.Components, c.Name)
							}
							for j, u := range inc.IncidentUpdates {
								if j >= 10 {
									break
								}
								item2.Updates = append(item2.Updates, providerUpdate{Status: u.Status, Body: u.Body, CreatedAt: u.CreatedAt})
							}
							item.Incidents = append(item.Incidents, item2)
						}
					}
					return item
				}
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.ProbeURL, nil)
	if err != nil {
		item.Status = "unknown"
		item.Description = "无法构造探测请求"
		item.CheckedVia = "probe"
		return item
	}
	resp, err := client.Do(req)
	item.CheckedVia = "probe"
	if err != nil {
		item.Status = "outage"
		item.Description = "API 端点不可达"
		return item
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 500:
		item.Status = "degraded"
		item.Description = fmt.Sprintf("API 返回 HTTP %d", resp.StatusCode)
	case resp.StatusCode == 429:
		item.Status = "degraded"
		item.Description = "API 限流中"
	default:
		item.Status = "operational"
		item.Description = "API 端点可达（官方状态页未提供机器接口）"
	}
	return item
}

func (s *Server) fetchProviders(ctx context.Context) []modelStatusItem {
	items := make([]modelStatusItem, len(modelProviders))
	var wg sync.WaitGroup
	for i, p := range modelProviders {
		wg.Add(1)
		go func(i int, p modelProvider) {
			defer wg.Done()
			items[i] = probeModelProvider(ctx, p)
		}(i, p)
	}
	wg.Wait()
	return items
}

func (s *Server) handleModelStatus(w http.ResponseWriter, r *http.Request) {
	s.modelStatusMu.Lock()
	if s.modelStatus != nil && time.Since(s.modelStatusAt) < 60*time.Second {
		items := append([]modelStatusItem(nil), s.modelStatus...)
		at := s.modelStatusAt
		s.modelStatusMu.Unlock()
		s.attachProviderHistory(r.Context(), items)
		writeJSON(w, http.StatusOK, map[string]any{"providers": items, "checkedAt": at})
		return
	}
	s.modelStatusMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	items := s.fetchProviders(ctx)

	s.modelStatusMu.Lock()
	s.modelStatus = items
	s.modelStatusAt = time.Now().UTC()
	s.modelStatusMu.Unlock()

	s.attachProviderHistory(r.Context(), items)
	writeJSON(w, http.StatusOK, map[string]any{"providers": items, "checkedAt": time.Now().UTC()})
}

func (s *Server) attachProviderHistory(ctx context.Context, items []modelStatusItem) {
	since := time.Now().UTC().AddDate(0, 0, -89)
	rows, err := s.svc.Repo().ProviderStatusHistory(ctx, since)
	if err != nil || len(rows) == 0 {
		return
	}
	byProvider := aggregateProviderHistory(rows, since)
	for i := range items {
		items[i].History = byProvider[items[i].Key]
	}
}

func aggregateProviderHistory(rows []store.ProviderSnapshot, since time.Time) map[string][]statusDay {
	const days = 90
	rank := map[string]int{"operational": 0, "unknown": 1, "degraded": 2, "outage": 3}
	type agg struct{ total, ok int; worst string }
	byProvider := map[string]map[string]*agg{}
	for _, row := range rows {
		if byProvider[row.Provider] == nil {
			byProvider[row.Provider] = map[string]*agg{}
		}
		date := row.CheckedAt.UTC().Format("2006-01-02")
		a := byProvider[row.Provider][date]
		if a == nil {
			a = &agg{worst: "operational"}
			byProvider[row.Provider][date] = a
		}
		a.total++
		if row.Status == "operational" {
			a.ok++
		}
		if rank[row.Status] > rank[a.worst] {
			a.worst = row.Status
		}
	}
	out := map[string][]statusDay{}
	for provider, dates := range byProvider {
		series := make([]statusDay, 0, days)
		for i := 0; i < days; i++ {
			date := since.AddDate(0, 0, i).Format("2006-01-02")
			day := statusDay{Date: date, Status: "unknown"}
			if a := dates[date]; a != nil && a.total > 0 {
				day.Status = a.worst
				day.Uptime = float64(a.ok) / float64(a.total) * 100
			}
			series = append(series, day)
		}
		out[provider] = series
	}
	return out
}

// StartProviderStatusRecorder snapshots upstream provider status every ten
// minutes so providers without a Statuspage still get a history.
func (s *Server) StartProviderStatusRecorder(ctx context.Context) {
	record := func() {
		cctx, cancel := context.WithTimeout(ctx, 12*time.Second)
		defer cancel()
		items := s.fetchProviders(cctx)
		now := time.Now().UTC()
		_ = s.svc.Repo().PruneProviderStatus(cctx, now.AddDate(0, 0, -180))
		for _, it := range items {
			s.providerPrevMu.Lock()
			prev, seen := s.providerLast[it.Key]
			s.providerLast[it.Key] = it.Status
			s.providerPrevMu.Unlock()
			if seen && prev == "operational" && it.Status != "operational" {
				go s.alerts.notifyProviderIncident(context.Background(), it.Key, it.Status, it.Description)
			}
			_ = s.svc.Repo().RecordProviderStatus(cctx, store.ProviderSnapshot{
				Provider: it.Key, Status: it.Status, Detail: it.Description, CheckedAt: now,
			})
		}
	}
	record()
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				record()
			}
		}
	}()
}
