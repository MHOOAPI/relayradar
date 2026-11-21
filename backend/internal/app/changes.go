package app

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"relayradar/internal/store"
)

// Change is one item in the public change feed.
type Change struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`     // new_station | purity_alert | availability_drop | price_change | score_drop
	Severity    string `json:"severity"` // info | warn | critical
	StationSlug string `json:"stationSlug"`
	StationName string `json:"stationName"`
	ModelID     string `json:"modelId,omitempty"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	// Numeric payload for price_change events, used by threshold filters.
	DeltaPct float64   `json:"deltaPct,omitempty"`
	PriceCNY float64   `json:"priceCny,omitempty"`
	At       time.Time `json:"at"`
}

// buildChanges derives the feed from data we already measure: new listings,
// latest purity checks, 24h-vs-72h availability drops and price history.
func buildChanges(snap *store.Snapshot, history map[string][]HistoryPoint, en bool) []Change {
	tr := func(zh, eng string) string {
		if en {
			return eng
		}
		return zh
	}
	stations := map[int64]store.Station{}
	for _, st := range snap.Stations {
		stations[st.ID] = st
	}
	groups := map[int64]store.Group{}
	for _, g := range snap.Groups {
		groups[g.ID] = g
	}
	models := map[string]store.Model{}
	for _, m := range snap.Models {
		models[m.ID] = m
	}

	now := snap.TakenAt
	var out []Change

	for _, st := range snap.Stations {
		if now.Sub(st.ListedAt) <= 14*24*time.Hour {
			days := int(now.Sub(st.ListedAt).Hours() / 24)
			if days < 1 {
				days = 1
			}
			out = append(out, Change{
				ID: "new-" + st.Slug, Kind: "new_station", Severity: "info",
				StationSlug: st.Slug, StationName: st.Name, At: st.ListedAt,
				Title:  fmt.Sprintf(tr("%s 进入观察期", "%s entered the watch period"), st.Name),
				Detail: fmt.Sprintf(tr("上架 %d 天，建议单次充值不超过 ¥20 并先用临时 Key 验证", "Listed %d days ago; top up no more than ¥20 at a time and verify with a temporary key first"), days),
			})
		}
	}

	latest := map[int64]store.Purity{}
	for _, p := range snap.Purity {
		if cur, ok := latest[p.GroupID]; !ok || p.CheckedAt.After(cur.CheckedAt) {
			latest[p.GroupID] = p
		}
	}
	for gid, p := range latest {
		if p.Score >= 30 {
			continue
		}
		g, ok := groups[gid]
		if !ok {
			continue
		}
		st := stations[g.StationID]
		out = append(out, Change{
			ID: "purity-" + st.Slug + "-" + g.Slug, Kind: "purity_alert", Severity: "critical",
			StationSlug: st.Slug, StationName: st.Name, At: p.CheckedAt,
			Title:  fmt.Sprintf(tr("%s 纯度抽检异常", "%s purity check failed"), g.Name),
			Detail: fmt.Sprintf(tr("最新抽检 %d 分，疑似替代模型或签名被剥离", "Latest check scored %d: possible substitution or stripped signature"), p.Score),
		})
	}

	// Availability: last 24h vs the 72h before that.
	type win struct{ ok, total int }
	recent := map[int64]*win{}
	prev := map[int64]*win{}
	recentStart := now.Add(-24 * time.Hour)
	prevStart := now.Add(-96 * time.Hour)
	for _, p := range snap.Probes {
		sid := int64(0)
		if g, ok := groups[p.GroupID]; ok {
			sid = g.StationID
		}
		if sid == 0 || p.ProbedAt.Before(prevStart) {
			continue
		}
		m := prev
		if p.ProbedAt.After(recentStart) {
			m = recent
		}
		if m[sid] == nil {
			m[sid] = &win{}
		}
		m[sid].total++
		if p.Status == "ok" || p.Status == "slow" {
			m[sid].ok++
		}
	}
	for sid, r := range recent {
		pv := prev[sid]
		if pv == nil || r.total < 12 || pv.total < 24 {
			continue
		}
		rRate := float64(r.ok) / float64(r.total)
		pRate := float64(pv.ok) / float64(pv.total)
		drop := (pRate - rRate) * 100
		if drop < 5 {
			continue
		}
		st := stations[sid]
		out = append(out, Change{
			ID: "drop-" + st.Slug, Kind: "availability_drop", Severity: "warn",
			StationSlug: st.Slug, StationName: st.Name, At: now.Add(-time.Minute),
			Title:  fmt.Sprintf(tr("可用率下降 %.1f 个百分点", "Availability dropped %.1f points"), drop),
			Detail: fmt.Sprintf(tr("近 24 小时 %.1f%%，前 3 天 %.1f%%", "Last 24h %.1f%%, previous 3 days %.1f%%"), rRate*100, pRate*100),
		})
	}

	// Price movements: compare the last two snapshots per group+model.
	type key struct {
		GroupID int64
		ModelID string
	}
	points := map[key][]store.PricePoint{}
	for _, p := range snap.Prices {
		k := key{p.GroupID, p.ModelID}
		points[k] = append(points[k], p)
	}
	for k, pts := range points {
		if len(pts) < 2 {
			continue
		}
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].RecordedAt.Before(pts[j].RecordedAt) })
		prevP, lastP := pts[len(pts)-2], pts[len(pts)-1]
		if prevP.InputCNY == nil || lastP.InputCNY == nil || *prevP.InputCNY <= 0 {
			continue
		}
		delta := (*lastP.InputCNY - *prevP.InputCNY) / *prevP.InputCNY
		if math.Abs(delta) < 0.01 {
			continue
		}
		g, ok := groups[k.GroupID]
		if !ok {
			continue
		}
		st := stations[g.StationID]
		m, ok := models[k.ModelID]
		if !ok {
			continue
		}
		dir, sev := tr("上调", "raised"), "warn"
		if delta < 0 {
			dir, sev = tr("下调", "lowered"), "info"
		}
		groupName := g.Name
		if en {
			groupName = strings.ReplaceAll(groupName, "默认组", "default")
		}
		out = append(out, Change{
			ID:   fmt.Sprintf("price-%s-%s-%d", st.Slug, k.ModelID, lastP.RecordedAt.Unix()),
			Kind: "price_change", Severity: sev,
			StationSlug: st.Slug, StationName: st.Name, At: lastP.RecordedAt,
			ModelID:  k.ModelID,
			DeltaPct: delta * 100, PriceCNY: *lastP.InputCNY,
			Title:  fmt.Sprintf(tr("%s 价格%s", "%s price %s"), groupName, dir),
			Detail: fmt.Sprintf(tr("%s：输入 ¥%.2f → ¥%.2f（%+.1f%%）", "%s: input ¥%.2f → ¥%.2f (%+.1f%%)"), m.Name, *prevP.InputCNY, *lastP.InputCNY, delta*100),
		})
	}

	// 14-day score regression detected from the daily history series.
	for _, st := range snap.Stations {
		pts := history[st.Slug]
		if len(pts) < 14 {
			continue
		}
		first := pts[len(pts)-14].Score
		last := pts[len(pts)-1].Score
		drop := first - last
		if drop < 3 {
			continue
		}
		sev := "warn"
		if drop >= 6 {
			sev = "critical"
		}
		out = append(out, Change{
			ID: "score-" + st.Slug, Kind: "score_drop", Severity: sev,
			StationSlug: st.Slug, StationName: st.Name, At: now,
			Title:  fmt.Sprintf(tr("综合分 14 天下降 %.1f 分", "Score fell %.1f points in 14 days"), drop),
			Detail: fmt.Sprintf(tr("从 %.1f 降到 %.1f，可用性、速度或稳定性出现回退，建议复查分组", "From %.1f to %.1f: availability, speed or consistency regressed; review the groups"), first, last),
		})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	// Official vendors are not relay stations; keep them out of the change feed.
	kept := out[:0]
	for _, c := range out {
		if officialVendors[c.StationSlug] {
			continue
		}
		kept = append(kept, c)
	}
	out = kept
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}
