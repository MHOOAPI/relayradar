// Command stationsync pulls a New-API style station's pricing and writes a snapshot.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type pricingModel struct {
	ModelName  string   `json:"model_name"`
	QuotaType  int      `json:"quota_type"`
	ModelPrice float64  `json:"model_price"`
}

type stationSnapshot struct {
	Slug    string  `json:"slug"`
	Name    string  `json:"name"`
	BaseURL string  `json:"baseUrl"`
	Groups  []group `json:"groups"`
}

type group struct {
	Name   string  `json:"name"`
	Models []model `json:"models"`
}

type model struct {
	Name        string  `json:"name"`
	PriceUSD    float64 `json:"priceUsd,omitempty"`
	PriceCNYIn  float64 `json:"priceCnyIn,omitempty"`
	PriceCNYOut float64 `json:"priceCnyOut,omitempty"`
	Unit        string  `json:"unit"`
}

func main() {
	base := flag.String("base", "", "station base URL")
	slug := flag.String("slug", "", "station slug")
	out := flag.String("out", "internal/store/station_pricing.json", "output file")
	flag.Parse()
	if *base == "" || *slug == "" {
		fmt.Fprintln(os.Stderr, "usage: stationsync -base URL -slug SLUG")
		os.Exit(2)
	}
	key := os.Getenv("STATION_API_KEY")
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}

	snap := stationSnapshot{Slug: *slug, Name: *slug, BaseURL: *base}

	var pricing struct {
		Data []pricingModel `json:"data"`
	}
	if err := getJSON(client, *base+"/api/pricing", key, &pricing); err == nil {
		var models []model
		for _, m := range pricing.Data {
			if m.QuotaType != 1 || m.ModelPrice <= 0 {
				continue
			}
			models = append(models, model{Name: m.ModelName, PriceUSD: m.ModelPrice, Unit: "image"})
		}
		if len(models) > 0 {
			snap.Groups = append(snap.Groups, group{Name: "image", Models: models})
		}
	}

	texts := fetchTextPricing(client, *base, key)
	if len(texts) > 0 {
		snap.Groups = append(snap.Groups, group{Name: "text", Models: texts})
	}

	for i := range snap.Groups {
		sort.Slice(snap.Groups[i].Models, func(a, b int) bool { return snap.Groups[i].Models[a].Name < snap.Groups[i].Models[b].Name })
	}
	sort.Slice(snap.Groups, func(i, j int) bool { return snap.Groups[i].Name < snap.Groups[j].Name })
	b, _ := json.MarshalIndent(snap, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d groups to %s\n", len(snap.Groups), *out)
}

func fetchTextPricing(client *http.Client, base, key string) []model {
	var payload struct {
		Data struct {
			BillingExpr map[string]string `json:"billing_expr"`
		} `json:"data"`
	}
	if err := getJSON(client, base+"/api/ratio_config", key, &payload); err != nil {
		return nil
	}
	re := regexp.MustCompile(`p \* ([0-9.]+) \+ c \* ([0-9.]+)`)
	var out []model
	for name, expr := range payload.Data.BillingExpr {
		m := re.FindStringSubmatch(expr)
		if len(m) != 3 {
			continue
		}
		in, _ := strconv.ParseFloat(m[1], 64)
		o, _ := strconv.ParseFloat(m[2], 64)
		if in <= 0 {
			continue
		}
		out = append(out, model{Name: name, PriceCNYIn: in, PriceCNYOut: o, Unit: "text"})
	}
	return out
}

func getJSON(client *http.Client, url, key string, dst any) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
