package app

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"relayradar/internal/scoring"
	"relayradar/internal/store"
)

type GroupView struct {
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Service      string    `json:"service"`
	Multiplier   float64   `json:"multiplier"`
	Status       string    `json:"status"`
	LatencyMS    int       `json:"latencyMs"`
	Availability float64   `json:"availability"`
	Speed        float64   `json:"speed"`
	Consistency  float64   `json:"consistency"`
	Quality      float64   `json:"quality"`
	Samples      int       `json:"samples"`
	Scored       bool      `json:"scored"`
	Purity       float64   `json:"purity"`
	PurityTier   string    `json:"purityTier"`
	ModelCount   int       `json:"modelCount"`
	LastProbeAt  time.Time `json:"lastProbeAt"`
}

type ReachabilityView struct {
	Status    string    `json:"status"`
	LatencyMS int       `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
}

type StationView struct {
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Domain         string    `json:"domain"`
	BaseURL        string    `json:"baseUrl"`
	IconURL        string    `json:"iconUrl"`
	Tagline        string    `json:"tagline"`
	TaglineEN      string    `json:"taglineEn"`
	OfficialVendor bool      `json:"officialVendor"`
	Rank           int       `json:"rank"`
	RankScore      float64   `json:"rankScore"`
	ScoreMargin    float64   `json:"scoreMargin"`
	Availability   float64   `json:"availability"`
	Speed          float64   `json:"speed"`
	Consistency    float64   `json:"consistency"`
	Samples        int       `json:"samples"`
	ScoredGroups   int       `json:"scoredGroups"`
	UptimePct      float64   `json:"uptimePct"`
	Purity         float64   `json:"purity"`
	PurityTier     string    `json:"purityTier"`
	PurityMeasured bool      `json:"purityMeasured"`
	ModelsCount    int       `json:"modelsCount"`
	ModelIDs       []string  `json:"modelIds"`
	Reachability   *ReachabilityView `json:"reachability,omitempty"`
	ReviewsCount   int       `json:"reviewsCount"`
	RatingAvg      float64   `json:"ratingAvg"`
	GroupsCount    int       `json:"groupsCount"`
	FromPriceCNY   float64   `json:"fromPriceCny"`
	MinTopup       float64   `json:"minTopupCny"`
	Payments       []string  `json:"payments"`
	Invoice        bool      `json:"invoice"`
	RefundPolicy   string    `json:"refundPolicy"`
	Claimed        bool      `json:"claimed"`
	Services       []string  `json:"services"`
	Uptime         []int     `json:"uptime"`
	Ranked         bool      `json:"ranked"`
	NewDomain      bool      `json:"newDomain"`
	ListedAt       time.Time `json:"listedAt"`
	LastProbeAt    time.Time `json:"lastProbeAt"`
}

type StationModel struct {
	Model         store.Model `json:"model"`
	BestInputCNY  float64     `json:"bestInputCny"`
	BestOutputCNY float64     `json:"bestOutputCny"`
	BestCacheCNY  float64     `json:"bestCacheCny"`
	CheapestGroup string      `json:"cheapestGroup"`
	Groups        []string    `json:"groups"`
}

// officialVendors are first-party model/cloud providers, not relay stations.
// They stay in the data set (useful for official price comparisons) but are
// excluded from the relay directory.
var officialVendors = map[string]bool{
	"bailian":     true,
	"siliconflow": true,
	"qiniu":       true,
	"ppio":        true,
}

// stationTaglineEN is the English one-liner shown on /en pages.
var stationTaglineEN = map[string]string{
	"bailian":    "Alibaba Cloud enterprise line for Qwen and DeepSeek",
	"302ai":      "Global model aggregator with an enterprise gateway",
	"openrouter": "Multi-cloud routing and official resale",
	"aihubmix":   "Official resale aggregator with a compliance entity",
	"packyapi":   "Dedicated line for Claude Code",
	"siliconflow": "Domestic inference cluster, self-operated",
	"qiniu":      "Enterprise backbone with high availability",
	"ppio":       "Distributed GPU compute pool",
	"ohmygpt":    "Multi-region direct lines for OpenAI and Claude",
	"closeai":    "Azure / AWS enterprise resale",
	"weelinking": "High-availability failover with a discount group",
	"ofoxai":     "US-West CN2 accelerated native resale",
	"dmxapi":     "Enterprise direct lines for three native protocols",
	"apiyi":      "Low-price direct lines for Claude and GPT",
	"yunwu":      "Discount and official groups kept separate",
	"laozhang":   "Direct billing through overseas corporate cards",
	"n1nai":      "Low-latency global routing",
	"wolf-cheap": "Fictional negative example: fake signatures and a hidden 1.62x markup",
}

type StationDetail struct {
	Station  StationView    `json:"station"`
	Groups   []GroupView    `json:"groups"`
	Models   []StationModel `json:"models"`
	Reviews  []store.Review `json:"reviews"`
	Business map[string]any `json:"business"`
	History  []HistoryPoint `json:"history"`
	// Outbound30d is the number of guidance-page clicks in the last 30 days.
	Outbound30d int `json:"outbound30d"`
}

type ModelView struct {
	store.Model
	OffersCount  int     `json:"offersCount"`
	MinInputCNY  float64 `json:"minInputCny"`
	MinOutputCNY float64 `json:"minOutputCny"`
	BestStation  string  `json:"bestStation"`
	License      string  `json:"license"`
	SummaryEN    string  `json:"summaryEn"`
}

// modelSummaryEN is the English summary used by the /en model pages.
var modelSummaryEN = map[string]string{
	"claude-fable-5-2":    "Coding and long-horizon agent flagship with native thinking signatures and a 1M context window.",
	"claude-opus-4-8":     "Previous-generation flagship; still the most common baseline for relay verification.",
	"claude-opus-5-5":     "Stable production workhorse at half the price of Fable 5.2.",
	"claude-sonnet-5-5":   "High-throughput completion model for IDE and real-time use.",
	"gpt-6.1-terra":       "Low-cost batch model for classification and extraction.",
	"gpt-6.2-astra":       "Multi-file autonomous refactoring flagship with a native 1.05M window.",
	"gpt-6.2-sol":         "Fast productivity model released alongside Astra.",
	"gemini-4-carbon":     "Native multimodal reasoning flagship with aggressive implicit cache discounts.",
	"gemini-4-hydrogen":   "Low-latency generation model at a low unit price.",
	"grok-4-7":            "Supports parallel sub-agent reasoning.",
	"deepseek-v4-1-flash": "Open-weight efficiency model with high throughput on domestic compute.",
	"kimi-k3-1-pro":       "Open weights, stable on very long repository contexts.",
	"qwen4-max":           "Trillion-scale MoE; strong at Chinese structured extraction.",
}

// modelLicenses is demo metadata; move to the models table once it comes from
// a real catalog.
var modelLicenses = map[string]string{
	"claude-fable-5-2":    "proprietary",
	"claude-opus-4-8":     "proprietary",
	"claude-opus-5-5":     "proprietary",
	"claude-sonnet-5-5":   "proprietary",
	"gpt-6.1-terra":       "proprietary",
	"gpt-6.2-astra":       "proprietary",
	"gpt-6.2-sol":         "proprietary",
	"gemini-4-carbon":     "proprietary",
	"gemini-4-hydrogen":   "proprietary",
	"grok-4-7":            "proprietary",
	"deepseek-v4-1-flash": "mit",
	"kimi-k3-1-pro":       "modified-mit",
	"qwen4-max":           "apache-2.0",
}

// LicenseOf returns the model's license slug.
func LicenseOf(id string) string {
	if v, ok := modelLicenses[id]; ok {
		return v
	}
	return "proprietary"
}

type Offer struct {
	StationSlug  string  `json:"stationSlug"`
	StationName  string  `json:"stationName"`
	GroupSlug    string  `json:"groupSlug"`
	GroupName    string  `json:"groupName"`
	Multiplier   float64 `json:"multiplier"`
	InputCNY     float64 `json:"inputCny"`
	OutputCNY    float64 `json:"outputCny"`
	CacheReadCNY float64 `json:"cacheReadCny"`
	StationRank  int     `json:"stationRank"`
	PurityTier   string  `json:"purityTier"`
	Availability float64 `json:"availability"`
	Claimed      bool    `json:"claimed"`
}

// PriceChangeView is one observed price movement for a model at a station.
type PriceChangeView struct {
	StationSlug string    `json:"stationSlug"`
	StationName string    `json:"stationName"`
	GroupName   string    `json:"groupName"`
	OldInputCNY float64   `json:"oldInputCny"`
	NewInputCNY float64   `json:"newInputCny"`
	DeltaPct    float64   `json:"deltaPct"`
	At          time.Time `json:"at"`
}

type Dataset struct {
	Stations         []StationView
	BySlug           map[string]*StationView
	Groups           map[int64]*GroupView
	GroupsByStation  map[string][]GroupView
	Models           []ModelView
	ModelByID        map[string]*ModelView
	Offers           map[string][]Offer
	PriceChanges     map[string][]PriceChangeView
	PerStationModels map[string][]StationModel
	History          map[string][]HistoryPoint
	Changes          []Change
	ChangesEN        []Change
	TakenAt          time.Time
}

type Service struct {
	st           store.Repo
	faviconProxy bool

	mu      sync.RWMutex
	data    *Dataset
	builtAt time.Time
}

func NewService(st store.Repo, faviconProxy bool) *Service {
	return &Service{st: st, faviconProxy: faviconProxy}
}

func (s *Service) Repo() store.Repo { return s.st }

// Invalidate drops the cached dataset so the next request rebuilds it.
func (s *Service) Invalidate() {
	s.mu.Lock()
	s.data = nil
	s.mu.Unlock()
}

func (s *Service) Dataset(ctx context.Context) (*Dataset, error) {
	s.mu.RLock()
	if s.data != nil && time.Since(s.builtAt) < 30*time.Second {
		d := s.data
		s.mu.RUnlock()
		return d, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data != nil && time.Since(s.builtAt) < 30*time.Second {
		return s.data, nil
	}

	snap, err := s.st.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	d := build(snap, s.faviconProxy)
	if reviews, err := s.st.ListAllReviews(ctx); err == nil && len(reviews) > 0 {
		counts := map[string]int{}
		sums := map[string]int{}
		for _, r := range reviews {
			counts[r.StationSlug]++
			sums[r.StationSlug] += r.Rating
		}
		for i := range d.Stations {
			slug := d.Stations[i].Slug
			if c := counts[slug]; c > 0 {
				d.Stations[i].ReviewsCount = c
				d.Stations[i].RatingAvg = math.Round(float64(sums[slug])/float64(c)*10) / 10
			}
		}
	}
	s.data = d
	s.builtAt = time.Now()
	if reach, err := s.st.ListReachability(ctx); err == nil {
		for i := range d.Stations {
			r, ok := reach[d.Stations[i].Slug]
			if !ok {
				continue
			}
			d.Stations[i].Reachability = &ReachabilityView{Status: r.Status, LatencyMS: r.LatencyMS, CheckedAt: r.CheckedAt}
			if v, ok := d.BySlug[r.Slug]; ok {
				v.Reachability = d.Stations[i].Reachability
			}
		}
	}
	return d, nil
}

func build(snap *store.Snapshot, faviconProxy bool) *Dataset {
	d := &Dataset{
		BySlug:           map[string]*StationView{},
		Groups:           map[int64]*GroupView{},
		GroupsByStation:  map[string][]GroupView{},
		ModelByID:        map[string]*ModelView{},
		Offers:           map[string][]Offer{},
		PriceChanges:     map[string][]PriceChangeView{},
		PerStationModels: map[string][]StationModel{},
		TakenAt:          snap.TakenAt,
	}

	probesByGroup := map[int64][]store.Probe{}
	for _, p := range snap.Probes {
		probesByGroup[p.GroupID] = append(probesByGroup[p.GroupID], p)
	}
	purityByGroup := map[int64][]store.Purity{}
	for _, p := range snap.Purity {
		if p.CheckedAt.After(snap.TakenAt.AddDate(0, 0, -7)) {
			purityByGroup[p.GroupID] = append(purityByGroup[p.GroupID], p)
		}
	}
	modelsByID := map[string]store.Model{}
	for _, m := range snap.Models {
		modelsByID[m.ID] = m
	}

	groupStation := map[int64]int64{}
	for _, g := range snap.Groups {
		groupStation[g.ID] = g.StationID
	}
	// Daily probe buckets for the 14-day uptime strip.
	type bucket struct{ ok, total int }
	dayStats := map[int64]map[string]*bucket{}
	rawStats := map[int64]*bucket{}
	for _, p := range snap.Probes {
		sid := groupStation[p.GroupID]
		if dayStats[sid] == nil {
			dayStats[sid] = map[string]*bucket{}
		}
		key := p.ProbedAt.UTC().Format("2006-01-02")
		b := dayStats[sid][key]
		if b == nil {
			b = &bucket{}
			dayStats[sid][key] = b
		}
		if rawStats[sid] == nil {
			rawStats[sid] = &bucket{}
		}
		b.total++
		rawStats[sid].total++
		if p.Status == "ok" || p.Status == "slow" {
			b.ok++
			rawStats[sid].ok++
		}
	}

	stationByID := map[int64]*StationView{}
	for i := range snap.Stations {
		st := snap.Stations[i]
		icon := st.IconURL
		if icon == "" && faviconProxy {
			icon = "/api/v1/favicon/" + st.Slug
		}
		v := &StationView{
			Slug: st.Slug, Name: st.Name, Domain: st.Domain, BaseURL: st.BaseURL, IconURL: icon, Tagline: st.Tagline, TaglineEN: stationTaglineEN[st.Slug], OfficialVendor: officialVendors[st.Slug],
			Payments: st.Payments, Invoice: st.Invoice, RefundPolicy: st.RefundPolicy,
			Claimed: st.Claimed, MinTopup: st.MinTopupCNY, ListedAt: st.ListedAt,
			NewDomain: snap.TakenAt.Sub(st.ListedAt) < 60*24*time.Hour,
			FromPriceCNY: -1,
		}
		d.BySlug[st.Slug] = v
		stationByID[st.ID] = v
		d.Stations = append(d.Stations, *v)
	}

	modelsByStation := map[int64]map[string]bool{}
	bestByStation := map[int64]float64{}

	var groupAxis = map[int64]scoring.Axis{}
	var serviceSet = map[int64]map[string]bool{}
	var groupCount = map[int64]int{}

	for _, g := range snap.Groups {
		probes := probesByGroup[g.ID]
		sevenDaysAgo := snap.TakenAt.AddDate(0, 0, -7)
		ok, slow, down := 0, 0, 0
		var latencies []int
		var lastAt time.Time
		var lastStatus string
		for _, p := range probes {
			if p.ProbedAt.After(lastAt) {
				lastAt = p.ProbedAt
				lastStatus = p.Status
			}
			if !p.ProbedAt.After(sevenDaysAgo) {
				continue
			}
			switch p.Status {
			case "ok":
				ok++
				latencies = append(latencies, p.LatencyMS)
			case "slow":
				slow++
				latencies = append(latencies, p.LatencyMS)
			default:
				down++
			}
		}
		total := ok + slow + down
		scored := total >= scoring.MinGroupSamples
		var axis scoring.Axis
		if scored {
			axis.Availability = scoring.WilsonLower(ok+slow, total)
			p50 := scoring.Percentile(latencies, 0.5)
			p95 := scoring.Percentile(latencies, 0.95)
			axis.Speed = scoring.SpeedScore(p50, len(latencies))
			axis.Consistency = scoring.ConsistencyScore(p50, p95, len(latencies))
		}

		var scores []int
		var purityScore float64
		measured := false
		for _, pc := range purityByGroup[g.ID] {
			scores = append(scores, pc.Score)
		}
		if len(scores) > 0 {
			measured = true
			purityScore = medianInts(scores)
		}

		mc := 0
		for _, gm := range snap.GroupModels {
			if gm.GroupID == g.ID {
				mc++
			}
		}

		gv := &GroupView{
			Slug: g.Slug, Name: g.Name, Service: g.Service, Multiplier: g.Multiplier,
			Status: lastStatus, LatencyMS: latencyOf(probes),
			Availability: scoring.Round1(axis.Availability),
			Speed:        scoring.Round1(axis.Speed),
			Consistency:  scoring.Round1(axis.Consistency),
			Quality:      scoring.Round1(scoring.Total(axis)),
			Samples:      total, Scored: scored,
			Purity: purityScore,
			PurityTier: scoring.TierOf(purityScore, measured), ModelCount: mc, LastProbeAt: lastAt,
		}
		d.Groups[g.ID] = gv
		if sv, ok := stationByID[g.StationID]; ok {
			d.GroupsByStation[sv.Slug] = append(d.GroupsByStation[sv.Slug], *gv)
		}
		groupAxis[g.ID] = axis
		groupCount[g.StationID]++
		if serviceSet[g.StationID] == nil {
			serviceSet[g.StationID] = map[string]bool{}
		}
		serviceSet[g.StationID][g.Service] = true
		if modelsByStation[g.StationID] == nil {
			modelsByStation[g.StationID] = map[string]bool{}
		}
		for _, gm := range snap.GroupModels {
			if gm.GroupID != g.ID {
				continue
			}
			modelsByStation[g.StationID][gm.ModelID] = true
			if gm.InputCNY != nil {
				cur, okk := bestByStation[g.StationID]
				if !okk || *gm.InputCNY < cur {
					bestByStation[g.StationID] = *gm.InputCNY
				}
			}
		}
	}

	// Per-station aggregation.
	for id, v := range stationByID {
		var inputs []scoring.GroupInput
		var purityAll []int
		var lastAt time.Time
		var samples int
		for _, g := range snap.Groups {
			if g.StationID != id {
				continue
			}
			gv := d.Groups[g.ID]
			samples += gv.Samples
			if gv.Scored {
				ax := groupAxis[g.ID]
				inputs = append(inputs, scoring.GroupInput{
					Weight:       float64(gv.ModelCount),
					Availability: ax.Availability,
					Speed:        ax.Speed,
					Consistency:  ax.Consistency,
				})
			}
			for _, pc := range purityByGroup[g.ID] {
				purityAll = append(purityAll, pc.Score)
			}
			if gv.LastProbeAt.After(lastAt) {
				lastAt = gv.LastProbeAt
			}
		}
		axis := scoring.Aggregate(inputs)
		v.RankScore = scoring.Round1(scoring.Total(axis))
		v.Availability = scoring.Round1(axis.Availability)
		v.Speed = scoring.Round1(axis.Speed)
		v.Consistency = scoring.Round1(axis.Consistency)
		v.Samples = samples
		v.ScoredGroups = len(inputs)
		v.GroupsCount = groupCount[id]
		v.ModelsCount = len(modelsByStation[id])
		ids := make([]string, 0, v.ModelsCount)
		for mid := range modelsByStation[id] {
			ids = append(ids, mid)
		}
		sort.Strings(ids)
		v.ModelIDs = ids
		v.LastProbeAt = lastAt
		if p, ok := bestByStation[id]; ok {
			v.FromPriceCNY = p
		}
		if len(purityAll) > 0 {
			v.Purity = medianInts(purityAll)
			v.PurityTier = scoring.TierOf(v.Purity, true)
			v.PurityMeasured = true
		} else {
			v.PurityTier = "unknown"
		}
		for svc := range serviceSet[id] {
			v.Services = append(v.Services, svc)
		}
		if v.Services == nil {
			v.Services = []string{}
		}
		sort.Strings(v.Services)

		v.Ranked = v.ScoredGroups > 0 && v.ScoredGroups*2 >= v.GroupsCount

		v.Uptime = make([]int, 14)
		if b := rawStats[id]; b != nil && b.total > 0 {
			v.UptimePct = scoring.Round1(float64(b.ok) / float64(b.total) * 100)
			// Only the availability sampling error is propagated, weighted by
			// its share of the total score. Speed/consistency remain point
			// estimates; the UI labels this explicitly.
			v.ScoreMargin = scoring.Round1(scoring.WilsonHalfWidth(b.ok, b.total) * 0.62)
		}
		for i := 0; i < 14; i++ {
			day := snap.TakenAt.AddDate(0, 0, -(13 - i)).UTC().Format("2006-01-02")
			if b, ok := dayStats[id][day]; ok && b.total > 0 {
				v.Uptime[i] = int(scoring.Round1(float64(b.ok) / float64(b.total) * 100))
			} else {
				v.Uptime[i] = -1
			}
		}
	}

	// Ranking.
	ranked := make([]*StationView, 0, len(stationByID))
	unranked := make([]*StationView, 0)
	for _, v := range stationByID {
		if v.Ranked {
			ranked = append(ranked, v)
		} else {
			unranked = append(unranked, v)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].RankScore == ranked[j].RankScore {
			if ranked[i].Availability == ranked[j].Availability {
				return ranked[i].Consistency > ranked[j].Consistency
			}
			return ranked[i].Availability > ranked[j].Availability
		}
		return ranked[i].RankScore > ranked[j].RankScore
	})
	for i, v := range ranked {
		v.Rank = i + 1
	}
	for _, v := range unranked {
		v.Rank = 0
	}

	d.Stations = make([]StationView, 0, len(stationByID))
	for _, v := range ranked {
		d.Stations = append(d.Stations, *v)
	}
	sort.SliceStable(unranked, func(i, j int) bool { return unranked[i].Name < unranked[j].Name })
	for _, v := range unranked {
		d.Stations = append(d.Stations, *v)
	}
	// Relay-only presentation: official vendors stay reachable by slug for
	// price comparison, but are not ranked as relay stations.
	relays := make([]StationView, 0, len(d.Stations))
	rank := 0
	for _, st := range d.Stations {
		if officialVendors[st.Slug] {
			continue
		}
		rank++
		st.Rank = rank
		relays = append(relays, st)
	}
	d.Stations = relays
	for _, v := range stationByID {
		cp := *v
		d.BySlug[v.Slug] = &cp
	}
	for slug := range d.GroupsByStation {
		gs := d.GroupsByStation[slug]
		sort.SliceStable(gs, func(i, j int) bool { return gs[i].Service < gs[j].Service })
		d.GroupsByStation[slug] = gs
	}

	// Models and offers.
	for _, m := range snap.Models {
		mv := ModelView{Model: m, MinInputCNY: -1, License: LicenseOf(m.ID), SummaryEN: modelSummaryEN[m.ID]}
		d.ModelByID[m.ID] = &mv
		d.Models = append(d.Models, mv)
	}
	stationMeta := map[int64]store.Station{}
	for _, st := range snap.Stations {
		stationMeta[st.ID] = st
	}

	for _, gm := range snap.GroupModels {
		g, ok := d.Groups[gm.GroupID]
		if !ok {
			continue
		}
		st, ok := stationMeta[groupStation[gm.GroupID]]
		if !ok {
			continue
		}
		stSlug, stName := st.Slug, st.Name
		var stRank int
		if sv, ok := d.BySlug[st.Slug]; ok {
			stRank = sv.Rank
		}
		if stSlug == "" || officialVendors[stSlug] {
			continue
		}
		o := Offer{
			StationSlug: stSlug, StationName: stName, GroupSlug: g.Slug, GroupName: g.Name,
			Multiplier: g.Multiplier, StationRank: stRank, PurityTier: g.PurityTier,
			Availability: g.Availability, Claimed: st.Claimed,
		}
		if gm.InputCNY != nil {
			o.InputCNY = *gm.InputCNY
		}
		if gm.OutputCNY != nil {
			o.OutputCNY = *gm.OutputCNY
		}
		if gm.CacheReadCNY != nil {
			o.CacheReadCNY = *gm.CacheReadCNY
		}
		d.Offers[gm.ModelID] = append(d.Offers[gm.ModelID], o)
		if mv, ok := d.ModelByID[gm.ModelID]; ok {
			// Counterfeit groups stay visible for transparency but never win the
			// "lowest price" slot.
			if g.PurityTier != "counterfeit" && (mv.MinInputCNY < 0 || (o.InputCNY > 0 && o.InputCNY < mv.MinInputCNY)) {
				mv.MinInputCNY = o.InputCNY
				mv.MinOutputCNY = o.OutputCNY
				mv.BestStation = stName
			}
			mv.OffersCount++
		}
	}
	for id, offers := range d.Offers {
		sort.SliceStable(offers, func(i, j int) bool {
			ci := offers[i].PurityTier == "counterfeit"
			cj := offers[j].PurityTier == "counterfeit"
			if ci != cj {
				return !ci
			}
			if offers[i].InputCNY == offers[j].InputCNY {
				return offers[i].StationRank < offers[j].StationRank
			}
			return offers[i].InputCNY < offers[j].InputCNY
		})
		d.Offers[id] = offers
	}
	for i := range d.Models {
		d.Models[i] = *d.ModelByID[d.Models[i].ID]
	}
	sort.SliceStable(d.Models, func(i, j int) bool { return d.Models[i].ReleasedAt.After(d.Models[j].ReleasedAt) })

	// Station model rows.
	for _, st := range snap.Stations {
		if officialVendors[st.Slug] {
			continue
		}
		perModel := map[string]*StationModel{}
		for _, g := range snap.Groups {
			if g.StationID != st.ID {
				continue
			}
			gv := d.Groups[g.ID]
			for _, gm := range snap.GroupModels {
				if gm.GroupID != g.ID {
					continue
				}
				row, ok := perModel[gm.ModelID]
				if !ok {
					m := modelsByID[gm.ModelID]
					row = &StationModel{Model: m, BestInputCNY: -1}
					perModel[gm.ModelID] = row
				}
				if gm.InputCNY != nil && (row.BestInputCNY < 0 || *gm.InputCNY < row.BestInputCNY) {
					row.BestInputCNY = *gm.InputCNY
					row.BestOutputCNY = deref(gm.OutputCNY)
					row.BestCacheCNY = deref(gm.CacheReadCNY)
					row.CheapestGroup = gv.Name
				}
				row.Groups = appendUnique(row.Groups, gv.Service)
			}
		}
		rows := make([]StationModel, 0, len(perModel))
		for _, row := range perModel {
			rows = append(rows, *row)
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Model.Vendor == rows[j].Model.Vendor {
				return rows[i].Model.Name < rows[j].Model.Name
			}
			return rows[i].Model.Vendor < rows[j].Model.Vendor
		})
		d.PerStationModels[st.Slug] = rows
	}

	d.History = buildHistory(snap)
	d.PriceChanges = buildPriceChanges(snap)
	d.Changes = buildChanges(snap, d.History, false)
	d.ChangesEN = buildChanges(snap, d.History, true)

	return d
}

func medianInts(v []int) float64 {
	if len(v) == 0 {
		return 0
	}
	cp := append([]int(nil), v...)
	sort.Ints(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 0 {
		return float64(cp[mid-1]+cp[mid]) / 2
	}
	return float64(cp[mid])
}

func latencyOf(probes []store.Probe) int {
	var l []int
	for _, p := range probes {
		if p.Status == "ok" && p.LatencyMS > 0 {
			l = append(l, p.LatencyMS)
		}
	}
	if len(l) == 0 {
		return 0
	}
	return int(scoring.Percentile(l, 0.5))
}

func deref(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

// Filter applies the list-page query.
func Filter(items []StationView, q, service, tier, sortKey string) []StationView {
	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]StationView, 0, len(items))
	for _, it := range items {
		if q != "" && !strings.Contains(strings.ToLower(it.Name), q) &&
			!strings.Contains(strings.ToLower(it.Domain), q) &&
			!strings.Contains(strings.ToLower(it.Tagline), q) {
			continue
		}
		if service != "" && service != "all" {
			found := false
			for _, s := range it.Services {
				if s == service {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if tier != "" && tier != "all" && it.PurityTier != tier {
			continue
		}
		out = append(out, it)
	}
	switch sortKey {
	case "price":
		sort.SliceStable(out, func(i, j int) bool {
			a, b := out[i].FromPriceCNY, out[j].FromPriceCNY
			if a < 0 {
				return false
			}
			if b < 0 {
				return true
			}
			return a < b
		})
	case "availability":
		sort.SliceStable(out, func(i, j int) bool { return out[i].UptimePct > out[j].UptimePct })
	case "speed":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Speed > out[j].Speed })
	case "models":
		sort.SliceStable(out, func(i, j int) bool { return out[i].ModelsCount > out[j].ModelsCount })
	}
	return out
}
