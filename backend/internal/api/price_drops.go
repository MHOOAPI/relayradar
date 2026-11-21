package api

import (
	"net/http"
	"sort"
	"strconv"
	"time"
)

type priceDrop struct {
	ModelID     string    `json:"modelId"`
	ModelName   string    `json:"modelName"`
	VendorName  string    `json:"vendorName"`
	StationSlug string    `json:"stationSlug"`
	StationName string    `json:"stationName"`
	OldInputCNY float64   `json:"oldInputCny"`
	NewInputCNY float64   `json:"newInputCny"`
	DeltaPct    float64   `json:"deltaPct"`
	At          time.Time `json:"at"`
}

// handlePriceDrops ranks the largest price cuts inside the window by comparing
// each station/model pair's first and last daily snapshot.
func (s *Server) handlePriceDrops(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	q := r.URL.Query()
	days := intParam(q.Get("days"), 30)
	if days < 1 || days > 180 {
		days = 30
	}
	limit := intParam(q.Get("limit"), 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	rows, err := s.svc.Repo().PriceDrops(r.Context(), days)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]priceDrop, 0, len(rows))
	for _, row := range rows {
		model, ok := d.ModelByID[row.ModelID]
		if !ok {
			continue
		}
		station, ok := d.BySlug[row.StationSlug]
		if !ok {
			continue
		}
		items = append(items, priceDrop{
			ModelID:     row.ModelID,
			ModelName:   model.Name,
			VendorName:  model.VendorName,
			StationSlug: row.StationSlug,
			StationName: station.Name,
			OldInputCNY: row.OldPriceCNY,
			NewInputCNY: row.NewPriceCNY,
			DeltaPct:    (row.NewPriceCNY - row.OldPriceCNY) / row.OldPriceCNY * 100,
			At:          row.NewDay,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].DeltaPct != items[j].DeltaPct {
			return items[i].DeltaPct < items[j].DeltaPct
		}
		return items[i].At.After(items[j].At)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	writeJSONCached(w, r, etagOf("price-drops", strconv.Itoa(days), strconv.Itoa(limit), strconv.Itoa(len(items)), d.TakenAt.String()), map[string]any{
		"days":  days,
		"items": items,
	})
}
