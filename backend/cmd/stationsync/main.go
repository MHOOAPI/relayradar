// Command stationsync pulls a New-API style station's public pricing and
// writes a snapshot used to seed real offers.
//
//	STATION_API_KEY=... go run ./cmd/stationsync -base https://api.mhoo.cc -slug mhoo
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type pricingModel struct {
	ModelName   string   `json:"model_name"`
	Description string   `json:"description"`
	QuotaType   int      `json:"quota_type"`
	ModelRatio  float64  `json:"model_ratio"`
	ModelPrice  float64  `json:"model_price"`
	Completion  float64  `json:"completion_ratio"`
	Groups      []string `json:"enable_groups"`
}

type stationSnapshot struct {
	Slug     string    `json:"slug"`
	Name     string    `json:"name"`
	BaseURL  string    `json:"baseUrl"`
	SyncedAt time.Time `json:"syncedAt"`
	Groups   []group   `json:"groups"`
}

type group struct {
	Name   string  `json:"name"`
	Models []model `json:"models"`
}

type model struct {
	Name    string  `json:"name"`
	PriceUSD float64 `json:"priceUsd"`
	Unit    string  `json:"unit"`
}

func main() {
	base := flag.String("base", "", "station base URL")
	slug := flag.String("slug", "", "station slug")
	out := flag.String("out", "internal/store/station_pricing.json", "output file")
	flag.Parse()
	if *base == "" || *slug == "" {
		fmt.Fprintln(os.Stderr, "usage: stationsync -base URL -slug SLUG (STATION_API_KEY env required)")
		os.Exit(2)
	}
	key := os.Getenv("STATION_API_KEY")
	req, _ := http.NewRequest(http.MethodGet, *base+"/api/pricing", nil)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var payload struct {
		Data []pricingModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(1)
	}

	byGroup := map[string][]model{}
	for _, m := range payload.Data {
		if m.QuotaType != 1 || m.ModelPrice <= 0 {
			continue
		}
		g := "默认组"
		if len(m.Groups) > 0 {
			g = m.Groups[0]
		}
		byGroup[g] = append(byGroup[g], model{Name: m.ModelName, PriceUSD: m.ModelPrice, Unit: "image"})
	}
	snap := stationSnapshot{Slug: *slug, Name: *slug, BaseURL: *base, SyncedAt: time.Now().UTC()}
	for name, models := range byGroup {
		snap.Groups = append(snap.Groups, group{Name: name, Models: models})
	}
	b, _ := json.MarshalIndent(snap, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d groups to %s\n", len(snap.Groups), *out)
}
