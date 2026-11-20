// Command prober sends real probes to the relays you own keys for.
//
//	prober -config probes.json -once
//	prober -config probes.json -interval 10m
//
// probes.json:
//
//	{"stations":[{"slug":"lingxi","service":"claude","baseUrl":"https://lingxi-api.com/v1","apiKey":"sk-...","model":"claude-opus-5-5"}]}
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"relayradar/internal/probe"
	"relayradar/internal/store"
)

func main() {
	configPath := flag.String("config", "probes.json", "path to probe credentials")
	once := flag.Bool("once", false, "run one round and exit")
	interval := flag.Duration("interval", 10*time.Minute, "delay between rounds")
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "PostgreSQL DSN")
	validate := flag.Bool("validate", false, "validate probes.json and exit without probing")
	apiURL := flag.String("api", "", "POST results to this API instead of writing the database directly")
	apiToken := flag.String("token", os.Getenv("PROBE_INGEST_TOKEN"), "ingest token for -api mode")
	flag.Parse()

	if *dsn == "" && !*validate && *apiURL == "" {
		log.Fatal("DATABASE_URL or -dsn is required")
	}
	raw, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var cfg probe.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}
	if len(cfg.Stations) == 0 {
		log.Fatal("no stations configured")
	}
	if *validate {
		for _, t := range cfg.Stations {
			log.Printf("target ok: %s/%s model=%s base=%s key=%t", t.Slug, t.Service, t.Model, t.BaseURL, t.APIKey != "")
		}
		log.Printf("configuration valid: %d targets", len(cfg.Stations))
		return
	}

	ctx := context.Background()
	var st *store.Store
	if *apiURL == "" {
		var err error
		st, err = store.Open(ctx, *dsn)
		if err != nil {
			log.Fatalf("db: %v", err)
		}
		defer st.Close()
	}

	client := &http.Client{Timeout: 20 * time.Second}

	postResult := func(t probe.Target, res probe.Result) error {
		body, _ := json.Marshal(map[string]any{
			"slug": t.Slug, "service": t.Service, "status": res.Status,
			"latencyMs": res.LatencyMS, "checkedAt": time.Now().UTC(),
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(*apiURL, "/")+"/api/v1/probes", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+*apiToken)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("ingest status %d", resp.StatusCode)
		}
		return nil
	}

	runRound := func() {
		for _, t := range cfg.Stations {
			res := probe.Run(ctx, t, client)
			if *apiURL != "" {
				if err := postResult(t, res); err != nil {
					log.Printf("%s/%s: ingest: %v", t.Slug, t.Service, err)
					continue
				}
				log.Printf("%s/%s -> %s (%dms) %s", t.Slug, t.Service, res.Status, res.LatencyMS, res.Detail)
				continue
			}
			groupID, err := st.GroupByStationService(ctx, t.Slug, t.Service)
			if err != nil {
				log.Printf("%s/%s: group not found: %v", t.Slug, t.Service, err)
				continue
			}
			if err := st.InsertProbe(ctx, groupID, res.Status, res.LatencyMS, time.Now().UTC()); err != nil {
				log.Printf("%s/%s: insert probe: %v", t.Slug, t.Service, err)
				continue
			}
			log.Printf("%s/%s -> %s (%dms) %s", t.Slug, t.Service, res.Status, res.LatencyMS, res.Detail)
		}
	}

	runRound()
	if *once {
		return
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for range ticker.C {
		runRound()
	}
}
