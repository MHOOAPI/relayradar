package api

import (
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"relayradar/internal/app"
	"relayradar/internal/store"
)

func (s *Server) handleAllReachabilityCSV(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.Repo().ListReachability(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	d, _ := s.svc.Dataset(r.Context())
	name := map[string]string{}
	if d != nil {
		for _, st := range d.Stations {
			name[st.Slug] = st.Name
		}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="relayradar-reachability.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"slug", "name", "status", "latency_ms", "checked_at"})
	for _, st := range d.Stations {
		if row, ok := rows[st.Slug]; ok {
			_ = cw.Write([]string{st.Slug, name[st.Slug], row.Status, fmt.Sprintf("%d", row.LatencyMS), row.CheckedAt.UTC().Format(time.RFC3339)})
		}
	}
	cw.Flush()
}

func (s *Server) handleStationReachability(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	since := time.Now().UTC().AddDate(0, 0, -89)
	rows, err := s.svc.Repo().ReachabilityHistory(r.Context(), slug, since)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	type agg struct {
		total int
		ok    int
		down  int
	}
	byDay := map[string]*agg{}
	for _, row := range rows {
		date := row.CheckedAt.UTC().Format("2006-01-02")
		a := byDay[date]
		if a == nil {
			a = &agg{}
			byDay[date] = a
		}
		a.total++
		switch row.Status {
		case "ok", "blocked":
			a.ok++
		case "down":
			a.down++
		}
	}
	days := make([]statusDay, 0, 90)
	for i := 0; i < 90; i++ {
		date := since.AddDate(0, 0, i).Format("2006-01-02")
		day := statusDay{Date: date, Status: "unknown"}
		if a := byDay[date]; a != nil && a.total > 0 {
			day.Uptime = float64(a.ok) / float64(a.total) * 100
			switch {
			case a.down > 0:
				day.Status = "outage"
			case a.ok == a.total:
				day.Status = "operational"
			default:
				day.Status = "degraded"
			}
		}
		days = append(days, day)
	}
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+slug+`-reachability.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"date", "status", "uptime_pct"})
		for _, d := range days {
			up := ""
			if d.Status != "unknown" {
				up = fmt.Sprintf("%.1f", d.Uptime)
			}
			_ = cw.Write([]string{d.Date, d.Status, up})
		}
		cw.Flush()
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"slug": slug, "days": days})
}


// stores the latest result. This is reachability only, not model quality.
func (s *Server) StartReachabilityRecorder(ctx context.Context) {
	if !s.cfg.ReachabilityEnabled {
		return
	}
	client := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Prefer IPv4: some relays hang on broken IPv6 routes.
				d := &net.Dialer{Timeout: 6 * time.Second}
				conn, err := d.DialContext(ctx, "tcp4", addr)
				if err == nil {
					return conn, nil
				}
				return d.DialContext(ctx, network, addr)
			},
		},
	}
	run := func() {
		d, err := s.svc.Dataset(ctx)
		if err != nil {
			return
		}
		rows := make([]store.Reachability, 0, len(d.Stations))
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 6)
		for _, st := range d.Stations {
			wg.Add(1)
			sem <- struct{}{}
			go func(st app.StationView) {
				defer wg.Done()
				defer func() { <-sem }()
				start := time.Now()
				status := "unknown"
				for _, method := range []string{http.MethodHead, http.MethodGet} {
					req, err := http.NewRequestWithContext(ctx, method, "https://"+st.Domain+"/", nil)
					if err != nil {
						continue
					}
					req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RelayRadar-Reachability/1.0)")
					req.Header.Set("Accept", "text/html,*/*")
					if resp, err := client.Do(req); err == nil {
						resp.Body.Close()
						switch {
						case resp.StatusCode < 400:
							status = "ok"
						case resp.StatusCode == 403 || resp.StatusCode == 405 || resp.StatusCode == 429:
							// Reachable, but refuses automated checks.
							status = "blocked"
						default:
							status = "down"
						}
						break
					}
				}
				mu.Lock()
				rows = append(rows, store.Reachability{
					Slug: st.Slug, Status: status, LatencyMS: int(time.Since(start).Milliseconds()), CheckedAt: time.Now().UTC(),
				})
				mu.Unlock()
			}(st)
		}
		wg.Wait()
		err = s.svc.Repo().RecordReachability(ctx, rows)
		if err == nil {
			s.svc.Invalidate()
		}
		for _, r := range rows {
			s.reachMu.Lock()
			if r.Status == "down" {
				s.reachDown[r.Slug]++
				if s.reachDown[r.Slug] == s.cfg.ReachabilityDownThreshold {
					go s.alerts.notifyStationDown(context.Background(), r.Slug, r.LatencyMS)
				}
			} else {
				if s.reachDown[r.Slug] >= s.cfg.ReachabilityDownThreshold {
					go s.alerts.notifyStationRecovered(context.Background(), r.Slug, r.LatencyMS)
				}
				delete(s.reachDown, r.Slug)
			}
			s.reachMu.Unlock()
		}
		s.log.Info("reachability round", "stations", len(rows), "err", err)
	}
	go run()
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

// StartFaviconWarm prefetches station favicons once the API is up, so a large
// directory renders instantly instead of waiting on a hundred third-party
// hosts during the first page view.
func (s *Server) StartFaviconWarm(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
		d, err := s.svc.Dataset(ctx)
		if err != nil || len(d.Stations) == 0 {
			return
		}
		domains := make([]string, 0, len(d.Stations))
		for _, st := range d.Stations {
			domains = append(domains, st.Domain)
		}
		s.favicons.Warm(domains)
	}()
}