package api

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"relayradar/internal/app"
)

type searchStation struct {
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	Domain         string  `json:"domain"`
	IconURL        string  `json:"iconUrl"`
	Rank           int     `json:"rank"`
	RankScore      float64 `json:"rankScore"`
	FromPriceCNY   float64 `json:"fromPriceCny"`
	PurityTier     string  `json:"purityTier"`
	OfficialVendor bool    `json:"officialVendor"`
}

type searchModel struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Vendor      string  `json:"vendor"`
	VendorName  string  `json:"vendorName"`
	Category    string  `json:"category"`
	MinInputCNY float64 `json:"minInputCny"`
	OffersCount int     `json:"offersCount"`
}

// matchRank scores a haystack against the lowercase query: 0 exact, 1 prefix,
// 2 substring, 3 no match.
func matchRank(hay, q string) int {
	h := strings.ToLower(hay)
	switch {
	case h == q:
		return 0
	case strings.HasPrefix(h, q):
		return 1
	case strings.Contains(h, q):
		return 2
	default:
		return 3
	}
}

func searchDataset(d *app.Dataset, query string, limit int) ([]searchStation, []searchModel) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" || limit <= 0 {
		return []searchStation{}, []searchModel{}
	}

	stations := make([]searchStation, 0, 8)
	type scoredStation struct {
		item  searchStation
		score int
	}
	scored := make([]scoredStation, 0, len(d.Stations))
	for i := range d.Stations {
		st := &d.Stations[i]
		score := matchRank(st.Name, q)
		if score == 3 {
			score = matchRank(st.Domain, q)
		}
		if score == 3 {
			score = matchRank(st.Slug, q)
		}
		if score == 3 {
			continue
		}
		scored = append(scored, scoredStation{
			score: score,
			item: searchStation{
				Slug:           st.Slug,
				Name:           st.Name,
				Domain:         st.Domain,
				IconURL:        st.IconURL,
				Rank:           st.Rank,
				RankScore:      st.RankScore,
				FromPriceCNY:   st.FromPriceCNY,
				PurityTier:     st.PurityTier,
				OfficialVendor: st.OfficialVendor,
			},
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score < scored[j].score
		}
		ri, rj := scored[i].item.Rank, scored[j].item.Rank
		if ri == 0 {
			ri = 1 << 20
		}
		if rj == 0 {
			rj = 1 << 20
		}
		return ri < rj
	})
	for i := 0; i < len(scored) && i < limit; i++ {
		stations = append(stations, scored[i].item)
	}

	models := make([]searchModel, 0, 8)
	type scoredModel struct {
		item  searchModel
		score int
	}
	scoredM := make([]scoredModel, 0, len(d.Models))
	for i := range d.Models {
		m := &d.Models[i]
		score := matchRank(m.Name, q)
		if score == 3 {
			score = matchRank(m.ID, q)
		}
		if score == 3 {
			score = matchRank(m.VendorName, q)
		}
		if score == 3 {
			score = matchRank(m.Vendor, q)
		}
		if score == 3 {
			continue
		}
		scoredM = append(scoredM, scoredModel{
			score: score,
			item: searchModel{
				ID:          m.ID,
				Name:        m.Name,
				Vendor:      m.Vendor,
				VendorName:  m.VendorName,
				Category:    m.Category,
				MinInputCNY: m.MinInputCNY,
				OffersCount: m.OffersCount,
			},
		})
	}
	sort.SliceStable(scoredM, func(i, j int) bool {
		if scoredM[i].score != scoredM[j].score {
			return scoredM[i].score < scoredM[j].score
		}
		return scoredM[i].item.OffersCount > scoredM[j].item.OffersCount
	})
	for i := 0; i < len(scoredM) && i < limit; i++ {
		models = append(models, scoredM[i].item)
	}
	return stations, models
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	limit := intParam(q.Get("limit"), 8)
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	if len([]rune(query)) > 64 {
		query = string([]rune(query)[:64])
	}
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	stations, models := searchDataset(d, query, limit)
	writeJSONCached(w, r, etagOf("search", d.TakenAt.String(), query, strconv.Itoa(limit)), map[string]any{
		"query":    query,
		"stations": stations,
		"models":   models,
	})
}

func etagOf(parts ...string) string {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf(`"v1-%x"`, h.Sum64())
}

// writeJSONCached adds an ETag and answers 304 for a matching If-None-Match.
func writeJSONCached(w http.ResponseWriter, r *http.Request, etag string, v any) {
	w.Header().Set("ETag", etag)
	if inm := r.Header.Get("If-None-Match"); inm != "" {
		for _, candidate := range strings.Split(inm, ",") {
			if strings.TrimSpace(candidate) == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, v)
}
