package api

import (
	"testing"
	"time"

	"relayradar/internal/app"
	"relayradar/internal/store"
)

func TestAlertMatchesThresholds(t *testing.T) {
	price := app.Change{Kind: "price_change", DeltaPct: -12, PriceCNY: 0.42, ModelID: "openai--gpt-6.2-sol", At: time.Now()}
	other := app.Change{Kind: "availability_drop", At: time.Now()}

	cases := []struct {
		name string
		sub  store.Subscription
		c    app.Change
		want bool
	}{
		{"no thresholds passes", store.Subscription{}, price, true},
		{"delta below floor filtered", store.Subscription{MinDeltaPct: 20}, price, false},
		{"delta above floor passes", store.Subscription{MinDeltaPct: 10}, price, true},
		{"price above ceiling filtered", store.Subscription{MaxPriceCNY: 0.4}, price, false},
		{"price below ceiling passes", store.Subscription{MaxPriceCNY: 0.5}, price, true},
		{"missing price filtered when ceiling set", store.Subscription{MaxPriceCNY: 1}, app.Change{Kind: "price_change"}, false},
		{"other kinds ignore thresholds", store.Subscription{MinDeltaPct: 50, MaxPriceCNY: 0.01}, other, true},
		{"model filter matches", store.Subscription{Models: []string{"openai--gpt-6.2-sol"}}, price, true},
		{"model filter mismatch", store.Subscription{Models: []string{"anthropic--claude"}}, price, false},
		{"model filter drops events without model", store.Subscription{Models: []string{"openai--gpt-6.2-sol"}}, app.Change{Kind: "price_change"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := alertMatches(tc.sub, tc.c); got != tc.want {
				t.Fatalf("alertMatches = %v, want %v", got, tc.want)
			}
		})
	}
}
