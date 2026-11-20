package app

import (
	"time"

	"relayradar/internal/scoring"
	"relayradar/internal/store"
)

// HistoryPoint is one daily score snapshot, recomputed from the probe window
// that ended on Date (7-day lookback, same v2 formula as the live score).
type HistoryPoint struct {
	Date         string  `json:"date"`
	Score        float64 `json:"score"`
	Availability float64 `json:"availability"`
	Speed        float64 `json:"speed"`
	Consistency  float64 `json:"consistency"`
	Uptime       float64 `json:"uptime"`
}

func buildHistory(snap *store.Snapshot) map[string][]HistoryPoint {
	modelCount := map[int64]int{}
	for _, gm := range snap.GroupModels {
		modelCount[gm.GroupID]++
	}
	probesByGroup := map[int64][]store.Probe{}
	for _, p := range snap.Probes {
		probesByGroup[p.GroupID] = append(probesByGroup[p.GroupID], p)
	}
	groupsByStation := map[int64][]store.Group{}
	for _, g := range snap.Groups {
		groupsByStation[g.StationID] = append(groupsByStation[g.StationID], g)
	}

	out := map[string][]HistoryPoint{}
	for _, st := range snap.Stations {
		groups := groupsByStation[st.ID]
		points := make([]HistoryPoint, 0, 30)

		for i := 29; i >= 0; i-- {
			end := snap.TakenAt.AddDate(0, 0, -i)
			start := end.AddDate(0, 0, -7)
			dayStart := end.Add(-24 * time.Hour)

			var inputs []scoring.GroupInput
			var dayOK, dayTotal int
			var earliest time.Time

			for _, g := range groups {
				var n, k int
				var lats []int
				for _, p := range probesByGroup[g.ID] {
					if p.ProbedAt.After(start) && !p.ProbedAt.After(end) {
						n++
						if earliest.IsZero() || p.ProbedAt.Before(earliest) {
							earliest = p.ProbedAt
						}
						if p.Status == "ok" || p.Status == "slow" {
							k++
							if p.LatencyMS > 0 {
								lats = append(lats, p.LatencyMS)
							}
						}
					}
					if p.ProbedAt.After(dayStart) && !p.ProbedAt.After(end) {
						dayTotal++
						if p.Status == "ok" || p.Status == "slow" {
							dayOK++
						}
					}
				}
				if n < 24 {
					continue
				}
				var ax scoring.Axis
				ax.Availability = scoring.WilsonLower(k, n)
				p50 := scoring.Percentile(lats, 0.5)
				p95 := scoring.Percentile(lats, 0.95)
				ax.Speed = scoring.SpeedScore(p50, len(lats))
				ax.Consistency = scoring.ConsistencyScore(p50, p95, len(lats))
				w := modelCount[g.ID]
				if w == 0 {
					w = 1
				}
				inputs = append(inputs, scoring.GroupInput{
					Weight: float64(w), Availability: ax.Availability,
					Speed: ax.Speed, Consistency: ax.Consistency,
				})
			}

			if len(inputs) == 0 {
				continue
			}
			// Skip days whose 7-day window is only partially covered by data,
			// otherwise early points look artificially low.
			if earliest.IsZero() || earliest.After(start.Add(24*time.Hour)) {
				continue
			}
			axis := scoring.Aggregate(inputs)
			uptime := 0.0
			if dayTotal > 0 {
				uptime = scoring.Round1(float64(dayOK) / float64(dayTotal) * 100)
			}
			points = append(points, HistoryPoint{
				Date:         end.UTC().Format("2006-01-02"),
				Score:        scoring.Round1(scoring.Total(axis)),
				Availability: scoring.Round1(axis.Availability),
				Speed:        scoring.Round1(axis.Speed),
				Consistency:  scoring.Round1(axis.Consistency),
				Uptime:       uptime,
			})
		}
		out[st.Slug] = points
	}
	return out
}
