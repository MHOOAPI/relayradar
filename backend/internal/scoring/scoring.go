package scoring

import (
	"math"
	"sort"
)

// Methodology v2.
//
// A group is scored on three published axes, each 0-100:
//
//	Availability  Wilson 95% lower bound of (ok+slow)/n  — conservative with small samples
//	Speed         median latency: <=800ms = 100, >=5000ms = 0
//	Consistency   p95 vs p50 jitter: no spread = 100, p95 at 3x p50 = 0
//
// The station score is 0.62*A + 0.26*S + 0.12*C, where each axis is a
// downside-adjusted mean across the station's scored groups:
// 80% coverage-weighted mean + 20% worst group. Coverage weight is the number
// of models the group sells, so a 40-model group matters more than a 2-model
// one, while the worst group still drags the result.
const (
	Z95 = 1.96

	SpeedBestMS  = 800.0
	SpeedWorstMS = 5000.0

	WeightAvailability = 0.62
	WeightSpeed        = 0.26
	WeightConsistency  = 0.12

	DownsideWeight = 0.20

	// MinGroupSamples is the number of probes a group needs before it is scored.
	MinGroupSamples = 24
)

// Axis holds the three published sub-scores for a group or a station.
type Axis struct {
	Availability float64 `json:"availability"`
	Speed        float64 `json:"speed"`
	Consistency  float64 `json:"consistency"`
}

// WilsonLower returns the lower bound of the 95% Wilson score interval for a
// binomial success rate, expressed as 0-100. With 84/84 probes it is ~95.6,
// with 100/100 it is ~96.3: a small sample cannot claim a perfect score.
// WilsonHalfWidth returns half of the 95% Wilson interval in percentage
// points. It is the sampling uncertainty of the raw success rate.
func WilsonHalfWidth(success, total int) float64 {
	if total <= 0 {
		return 0
	}
	if success < 0 {
		success = 0
	}
	if success > total {
		success = total
	}
	n := float64(total)
	p := float64(success) / n
	z2 := Z95 * Z95
	denom := 1 + z2/n
	centre := p + z2/(2*n)
	margin := Z95 * math.Sqrt(p*(1-p)/n+z2/(4*n*n))
	lo := (centre - margin) / denom
	hi := (centre + margin) / denom
	return clamp(hi-lo, 0, 1) / 2 * 100
}

func WilsonLower(success, total int) float64 {
	if total <= 0 {
		return 0
	}
	if success < 0 {
		success = 0
	}
	if success > total {
		success = total
	}
	n := float64(total)
	p := float64(success) / n
	z2 := Z95 * Z95
	denom := 1 + z2/n
	centre := p + z2/(2*n)
	margin := Z95 * math.Sqrt(p*(1-p)/n+z2/(4*n*n))
	return clamp((centre-margin)/denom, 0, 1) * 100
}

// Percentile returns the q-quantile (0..1) of latencies with linear interpolation.
func Percentile(latencies []int, q float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	vals := append([]int(nil), latencies...)
	sort.Ints(vals)
	if len(vals) == 1 {
		return float64(vals[0])
	}
	q = clamp(q, 0, 1)
	pos := q * float64(len(vals)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return float64(vals[lo])
	}
	frac := pos - float64(lo)
	return float64(vals[lo])*(1-frac) + float64(vals[hi])*frac
}

// SpeedScore maps median latency onto 0-100. Too few samples yields a neutral 50.
func SpeedScore(p50 float64, samples int) float64 {
	if samples < 3 || p50 <= 0 {
		return 50
	}
	return 100 * clamp((SpeedWorstMS-p50)/(SpeedWorstMS-SpeedBestMS), 0, 1)
}

// ConsistencyScore penalises p95 spreading away from p50. Too few samples yields 60.
func ConsistencyScore(p50, p95 float64, samples int) float64 {
	if samples < 10 || p50 <= 0 {
		return 60
	}
	jitter := (p95 - p50) / math.Max(p50, 150)
	return 100 * clamp(1-jitter/2, 0, 1)
}

// GroupInput is one scored group contributing to a station.
type GroupInput struct {
	Weight                   float64
	Availability, Speed, Consistency float64
}

// Aggregate blends the 80% coverage-weighted mean with the 20% worst group.
func Aggregate(groups []GroupInput) Axis {
	if len(groups) == 0 {
		return Axis{}
	}
	var wsum, aW, sW, cW float64
	minA, minS, minC := 100.0, 100.0, 100.0
	for _, g := range groups {
		w := g.Weight
		if w <= 0 {
			w = 1
		}
		wsum += w
		aW += g.Availability * w
		sW += g.Speed * w
		cW += g.Consistency * w
		minA = math.Min(minA, g.Availability)
		minS = math.Min(minS, g.Speed)
		minC = math.Min(minC, g.Consistency)
	}
	blend := func(sum, min float64) float64 {
		mean := sum / wsum
		return (1-DownsideWeight)*mean + DownsideWeight*min
	}
	return Axis{
		Availability: blend(aW, minA),
		Speed:        blend(sW, minS),
		Consistency:  blend(cW, minC),
	}
}

// Total combines the three axes with the published weights.
func Total(a Axis) float64 {
	return WeightAvailability*a.Availability + WeightSpeed*a.Speed + WeightConsistency*a.Consistency
}

// TierOf buckets an authenticity score into the three public tiers.
func TierOf(score float64, measured bool) string {
	if !measured {
		return "unknown"
	}
	switch {
	case score >= 80:
		return "official"
	case score >= 30:
		return "mixed"
	default:
		return "counterfeit"
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func Round1(v float64) float64 { return math.Round(v*10) / 10 }
