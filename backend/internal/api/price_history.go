package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"relayradar/internal/store"
)

// StartPriceSnapshots records one price snapshot per day so the model pages can
// draw a price series even though the catalogue only exposes the current value.
func (s *Server) StartPriceSnapshots(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(20 * time.Second):
		}
		s.snapshotPrices(ctx)
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.snapshotPrices(ctx)
			}
		}
	}()
}

func (s *Server) snapshotPrices(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	n, err := s.svc.Repo().SnapshotPriceDaily(cctx, time.Now().UTC())
	s.snapMu.Lock()
	if err != nil {
		s.snapErr = err.Error()
	} else {
		s.snapLast = time.Now().UTC()
		s.snapRows = n
		s.snapErr = ""
	}
	s.snapMu.Unlock()
	if err != nil {
		s.log.Warn("price snapshot failed", "err", err)
		return
	}
	s.log.Info("price snapshot written", "rows", n)
}

type pricePoint struct {
	Day          string  `json:"day"`
	MinInputCNY  float64 `json:"minInputCny"`
	MinOutputCNY float64 `json:"minOutputCny"`
	Stations     int     `json:"stations"`
	Offers       int     `json:"offers"`
}

func pricePointsJSON(points []store.PriceDailyPoint) []pricePoint {
	out := make([]pricePoint, 0, len(points))
	for _, p := range points {
		out = append(out, pricePoint{p.Day.Format("2006-01-02"), p.MinInputCNY, p.MinOutputCNY, p.Stations, p.Offers})
	}
	return out
}

func (s *Server) handleModelPriceHistory(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	id := r.PathValue("id")
	days := intParam(r.URL.Query().Get("days"), 90)
	if days < 1 || days > 365 {
		days = 90
	}
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	if _, ok := d.ModelByID[id]; !ok {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	points, err := s.svc.Repo().PriceDailyHistory(r.Context(), id, days)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	out := pricePointsJSON(points)
	etag := etagOf("price-history", id, strconv.Itoa(days), strconv.Itoa(len(out)))
	writeJSONCached(w, r, etag, map[string]any{
		"modelId": id,
		"days":    days,
		"points":  out,
	})
}

func (s *Server) handleAdminPriceSnapshot(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	start := time.Now()
	n, err := s.svc.Repo().SnapshotPriceDaily(r.Context(), time.Now().UTC())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	s.snapMu.Lock()
	s.snapLast = time.Now().UTC()
	s.snapRows = n
	s.snapErr = ""
	s.snapMu.Unlock()
	s.logAdminAction(r, admin, "price.snapshot", "price_daily", fmt.Sprintf("rows=%d", n))
	writeJSON(w, http.StatusOK, map[string]any{
		"rows":       n,
		"durationMs": time.Since(start).Milliseconds(),
	})
}

func (s *Server) handleStationPriceHistory(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	slug := r.PathValue("slug")
	q := r.URL.Query()
	modelID := strings.TrimSpace(q.Get("model"))
	days := intParam(q.Get("days"), 90)
	if days < 1 || days > 365 {
		days = 90
	}
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	if _, ok := d.BySlug[slug]; !ok {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	if modelID == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}
	if _, ok := d.ModelByID[modelID]; !ok {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	points, err := s.svc.Repo().StationPriceHistory(r.Context(), slug, modelID, days)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	out := pricePointsJSON(points)
	etag := etagOf("station-price-history", slug, modelID, strconv.Itoa(days), strconv.Itoa(len(out)))
	writeJSONCached(w, r, etag, map[string]any{
		"stationSlug": slug,
		"modelId":     modelID,
		"days":        days,
		"points":      out,
	})
}
