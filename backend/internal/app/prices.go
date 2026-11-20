package app

import (
	"sort"

	"relayradar/internal/scoring"
	"relayradar/internal/store"
)

func buildPriceChanges(snap *store.Snapshot) map[string][]PriceChangeView {
	stations := map[int64]store.Station{}
	for _, st := range snap.Stations {
		stations[st.ID] = st
	}
	groups := map[int64]store.Group{}
	for _, g := range snap.Groups {
		groups[g.ID] = g
	}

	type key struct {
		GroupID int64
		ModelID string
	}
	points := map[key][]store.PricePoint{}
	for _, p := range snap.Prices {
		k := key{p.GroupID, p.ModelID}
		points[k] = append(points[k], p)
	}

	out := map[string][]PriceChangeView{}
	for k, pts := range points {
		if len(pts) < 2 {
			continue
		}
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].RecordedAt.Before(pts[j].RecordedAt) })
		g, ok := groups[k.GroupID]
		if !ok {
			continue
		}
		st, ok := stations[g.StationID]
		if !ok {
			continue
		}
		for i := 1; i < len(pts); i++ {
			prev, cur := pts[i-1], pts[i]
			if prev.InputCNY == nil || cur.InputCNY == nil || *prev.InputCNY <= 0 {
				continue
			}
			delta := (*cur.InputCNY - *prev.InputCNY) / *prev.InputCNY
			if delta > -0.01 && delta < 0.01 {
				continue
			}
			out[k.ModelID] = append(out[k.ModelID], PriceChangeView{
				StationSlug: st.Slug, StationName: st.Name, GroupName: g.Name,
				OldInputCNY: *prev.InputCNY, NewInputCNY: *cur.InputCNY,
				DeltaPct: scoring.Round1(delta * 100), At: cur.RecordedAt,
			})
		}
	}

	for id := range out {
		list := out[id]
		sort.SliceStable(list, func(i, j int) bool { return list[i].At.After(list[j].At) })
		if len(list) > 5 {
			list = list[:5]
		}
		out[id] = list
	}
	return out
}
