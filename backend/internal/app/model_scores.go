package app

import (
	"math"
	"sort"
)

// Model scores are derived only from what RelayRadar can observe: the offers a
// model has, their prices, their measured availability and purity, plus the
// context window from the catalog. They describe supply (how good this model is
// to buy right now), not benchmark capability, which we do not measure.

func clampPct(v float64) float64 {
	return math.Max(0, math.Min(100, v))
}

// computeSupplyScores fills the score fields on a ModelView from its offers.
func (m *ModelView) computeSupplyScores(offers []Offer) {
	var official, mixed, unknown, counterfeit int
	var availSum float64
	var availN int
	prices := make([]float64, 0, len(offers))
	low, high := 0.0, 0.0

	for _, o := range offers {
		switch o.PurityTier {
		case "official":
			official++
		case "mixed":
			mixed++
		case "counterfeit":
			counterfeit++
		default:
			unknown++
		}
		if o.Availability > 0 {
			availSum += o.Availability
			availN++
		}
		if o.InputCNY > 0 {
			prices = append(prices, o.InputCNY)
			if o.PurityTier != "counterfeit" && (low == 0 || o.InputCNY < low) {
				low = o.InputCNY
			}
			if high == 0 || o.InputCNY > high {
				high = o.InputCNY
			}
		}
	}

	m.PurityOfficial = official
	m.PurityMixed = mixed
	m.PurityUnknown = unknown
	m.PurityCounterfeit = counterfeit
	m.PriceLowCNY = low
	m.PriceHighCNY = high
	m.PriceMedianCNY = median(prices)
	if availN > 0 {
		m.AvgAvailability = round1(availSum / float64(availN))
	}

	total := len(offers)
	if total > 0 {
		m.PurityScore = round1(clampPct(
			(float64(official)*100 + float64(mixed)*65 + float64(unknown)*40) / float64(total)))
		m.CoverageScore = round1(clampPct(100 * math.Sqrt(float64(total)/50)))
		m.AvailabilityScore = round1(clampPct(m.AvgAvailability))
	}

	isText := m.Category == "" || m.Category == "text"
	if isText && m.ContextWindow >= 1000 {
		m.ContextScore = round1(clampPct(100 * math.Log10(float64(m.ContextWindow)) / 6))
	}

	// Price score measures how much cheaper the cheapest non-counterfeit offer
	// is than the median offer: a market where the cheapest is far below the
	// typical price is worth comparing, a flat market is not.
	if low > 0 && m.PriceMedianCNY > 0 {
		ratio := low / m.PriceMedianCNY
		m.PriceScore = round1(clampPct(100 * (1 - ratio/2)))
		m.PriceRatio = round2(ratio)
	}

	type axis struct {
		value  float64
		weight float64
		ok     bool
	}
	axes := []axis{
		{m.PriceScore, 0.35, low > 0 && m.PriceMedianCNY > 0},
		{m.CoverageScore, 0.20, total > 0},
		{m.AvailabilityScore, 0.20, availN > 0},
		{m.PurityScore, 0.15, total > 0},
		{m.ContextScore, 0.10, isText && m.ContextWindow >= 1000},
	}
	var sum, weight float64
	for _, a := range axes {
		if !a.ok {
			continue
		}
		sum += a.value * a.weight
		weight += a.weight
	}
	if weight > 0 {
		m.RelayScore = round1(sum / weight)
	}
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 0 {
		return round2((s[mid-1] + s[mid]) / 2)
	}
	return round2(s[mid])
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func round2(v float64) float64 { return math.Round(v*100) / 100 }
