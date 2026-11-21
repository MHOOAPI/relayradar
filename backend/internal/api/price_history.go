package api

import (
	"context"
	"net/http"
	"strconv"
	"time"
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
	if err != nil {
		s.log.Warn("price snapshot failed", "err", err)
		return
	}
	s.log.Info("price snapshot written", "rows", n)
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
	type point struct {
		Day          string  `json:"day"`
		MinInputCNY  float64 `json:"minInputCny"`
		MinOutputCNY float64 `json:"minOutputCny"`
		Stations     int     `json:"stations"`
		Offers       int     `json:"offers"`
	}
	out := make([]point, 0, len(points))
	for _, p := range points {
		out = append(out, point{p.Day.Format("2006-01-02"), p.MinInputCNY, p.MinOutputCNY, p.Stations, p.Offers})
	}
	etag := etagOf("price-history", id, strconv.Itoa(days), strconv.Itoa(len(out)))
	writeJSONCached(w, r, etag, map[string]any{
		"modelId": id,
		"days":    days,
		"points":  out,
	})
}
