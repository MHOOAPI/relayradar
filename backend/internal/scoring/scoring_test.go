package scoring

import (
	"math"
	"testing"
)

func near(a, b, eps float64) bool { return math.Abs(a-b) < eps }

func TestWilsonHalfWidth(t *testing.T) {
	if got := WilsonHalfWidth(0, 0); got != 0 {
		t.Fatalf("empty sample half width = %v, want 0", got)
	}
	small := WilsonHalfWidth(84, 84)
	large := WilsonHalfWidth(2500, 2500)
	if small <= large {
		t.Fatalf("small sample (%v) should be less certain than large (%v)", small, large)
	}
	if small < 0.5 || small > 5 {
		t.Fatalf("84/84 half width = %v, want roughly 1-2pp", small)
	}
}

func TestWilsonLower(t *testing.T) {
	if got := WilsonLower(0, 0); got != 0 {
		t.Fatalf("no samples = %v, want 0", got)
	}
	cases := []struct{ success, total int; want float64 }{
		{100, 100, 96.3},
		{84, 84, 95.6},
		{95, 100, 88.8},
		{70, 100, 60.4},
	}
	for _, c := range cases {
		got := WilsonLower(c.success, c.total)
		if !near(got, c.want, 0.2) {
			t.Fatalf("WilsonLower(%d,%d) = %.2f, want ~%.1f", c.success, c.total, got, c.want)
		}
	}
	if WilsonLower(100, 100) >= 100 {
		t.Fatal("small perfect sample must stay below 100")
	}
	if WilsonLower(90, 100) <= WilsonLower(80, 100) {
		t.Fatal("more successes must not lower the bound")
	}
}

func TestPercentile(t *testing.T) {
	lats := []int{100, 200, 300, 400, 500}
	if got := Percentile(lats, 0.5); got != 300 {
		t.Fatalf("p50 = %v, want 300", got)
	}
	if got := Percentile(lats, 0.95); got < 470 || got > 500 {
		t.Fatalf("p95 = %v, want near 480-500", got)
	}
	if got := Percentile(nil, 0.5); got != 0 {
		t.Fatalf("empty = %v, want 0", got)
	}
}

func TestSpeedScore(t *testing.T) {
	cases := []struct{ p50 float64; want float64 }{
		{800, 100},
		{5000, 0},
		{2900, 50},
	}
	for _, c := range cases {
		if got := SpeedScore(c.p50, 20); !near(got, c.want, 0.5) {
			t.Fatalf("SpeedScore(%v) = %v, want %v", c.p50, got, c.want)
		}
	}
	if got := SpeedScore(900, 2); got != 50 {
		t.Fatalf("too few samples should be neutral 50, got %v", got)
	}
}

func TestConsistencyScore(t *testing.T) {
	if got := ConsistencyScore(1000, 1000, 50); got != 100 {
		t.Fatalf("no jitter = %v, want 100", got)
	}
	if got := ConsistencyScore(1000, 3000, 50); got != 0 {
		t.Fatalf("p95 at 3x p50 = %v, want 0", got)
	}
	if got := ConsistencyScore(1000, 2000, 50); !near(got, 50, 0.5) {
		t.Fatalf("p95 at 2x p50 = %v, want 50", got)
	}
}

func TestAggregateDownside(t *testing.T) {
	// Equal weights, one excellent and one poor group: the poor one still
	// gets 20% of the final axis.
	axis := Aggregate([]GroupInput{
		{Weight: 1, Availability: 100, Speed: 100, Consistency: 100},
		{Weight: 1, Availability: 50, Speed: 50, Consistency: 50},
	})
	if !near(axis.Availability, 0.8*75+0.2*50, 0.01) {
		t.Fatalf("availability = %v, want 70", axis.Availability)
	}
	// Coverage weight must shift the mean toward the heavier group.
	axis = Aggregate([]GroupInput{
		{Weight: 9, Availability: 90, Speed: 90, Consistency: 90},
		{Weight: 1, Availability: 40, Speed: 40, Consistency: 40},
	})
	if axis.Availability >= 0.8*85+0.2*40+0.01 {
		t.Fatalf("coverage weighting not applied: %v", axis.Availability)
	}
	if got := Aggregate(nil); got != (Axis{}) {
		t.Fatalf("empty = %+v", got)
	}
}

func TestTotal(t *testing.T) {
	a := Axis{Availability: 90, Speed: 80, Consistency: 70}
	want := 0.62*90 + 0.26*80 + 0.12*70
	if got := Total(a); !near(got, want, 0.001) {
		t.Fatalf("Total = %v, want %v", got, want)
	}
	if Total(Axis{Availability: 100, Speed: 0, Consistency: 0}) != 62 {
		t.Fatal("availability-only axis should be 62")
	}
}

func TestTierOf(t *testing.T) {
	cases := map[float64]string{95: "official", 80: "official", 79.9: "mixed", 30: "mixed", 29.9: "counterfeit", 0: "counterfeit"}
	for score, want := range cases {
		if got := TierOf(score, true); got != want {
			t.Fatalf("TierOf(%v) = %q, want %q", score, got, want)
		}
	}
	if got := TierOf(99, false); got != "unknown" {
		t.Fatalf("unmeasured should be unknown, got %q", got)
	}
}
