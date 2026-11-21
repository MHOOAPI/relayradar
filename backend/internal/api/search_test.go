package api

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"relayradar/internal/app"
	"relayradar/internal/store"
)

func TestSearchDatasetRanksStations(t *testing.T) {
	d := &app.Dataset{
		Stations: []app.StationView{
			{Slug: "openrouter", Name: "OpenRouter", Domain: "openrouter.ai", Rank: 1},
			{Slug: "openai-hub", Name: "OpenAI Hub", Domain: "openaihub.dev", Rank: 5},
			{Slug: "cun", Name: "CUN.ai", Domain: "cun.ai", Rank: 40},
		},
	}
	stations, models := searchDataset(d, "openrouter", 5)
	if len(models) != 0 {
		t.Fatalf("unexpected models: %d", len(models))
	}
	if len(stations) == 0 || stations[0].Slug != "openrouter" {
		t.Fatalf("exact match not first: %+v", stations)
	}
	if len(stations) != 1 {
		t.Fatalf("expected only the openrouter match, got %d", len(stations))
	}

	stations, _ = searchDataset(d, "open", 5)
	if len(stations) != 2 || stations[0].Slug != "openrouter" {
		t.Fatalf("prefix ranking wrong: %+v", stations)
	}

	stations, _ = searchDataset(d, "cun.ai", 5)
	if len(stations) != 1 || stations[0].Slug != "cun" {
		t.Fatalf("domain search wrong: %+v", stations)
	}
}

func TestSearchDatasetEmptyQuery(t *testing.T) {
	d := &app.Dataset{Stations: []app.StationView{{Slug: "a", Name: "A"}}}
	stations, models := searchDataset(d, "   ", 5)
	if len(stations) != 0 || len(models) != 0 {
		t.Fatalf("empty query should return nothing, got %d/%d", len(stations), len(models))
	}
}

func TestSearchDatasetRanksModelsByOffers(t *testing.T) {
	d := &app.Dataset{
		Models: []app.ModelView{
			{Model: store.Model{ID: "openai--gpt-6.2-sol", Name: "GPT-6.2 Sol", Vendor: "openai"}, OffersCount: 100},
			{Model: store.Model{ID: "openai--gpt-6.2-sol-pro", Name: "GPT-6.2 Sol Pro", Vendor: "openai"}, OffersCount: 300},
			{Model: store.Model{ID: "anthropic--claude", Name: "Claude", Vendor: "anthropic"}, OffersCount: 50},
		},
	}
	_, models := searchDataset(d, "gpt-6.2", 5)
	if len(models) != 2 {
		t.Fatalf("expected 2 model matches, got %d", len(models))
	}
	if models[0].OffersCount != 300 {
		t.Fatalf("more offers should rank first: %+v", models)
	}
}

func TestWriteJSONCachedHonoursIfNoneMatch(t *testing.T) {
	etag := etagOf("stations", "2026-01-01T00:00:00Z", "42")
	rec := httptest.NewRecorder()
	writeJSONCached(rec, httptest.NewRequest("GET", "/api/v1/stations", nil), etag, map[string]string{"ok": "1"})
	if rec.Code != 200 || rec.Header().Get("ETag") != etag {
		t.Fatalf("first response: code=%d etag=%q", rec.Code, rec.Header().Get("ETag"))
	}
	if !strings.Contains(rec.Body.String(), `"ok":"1"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	req := httptest.NewRequest("GET", "/api/v1/stations", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	writeJSONCached(rec, req, etag, map[string]string{"ok": "1"})
	if rec.Code != 304 || rec.Body.Len() != 0 {
		t.Fatalf("expected 304 with empty body, got %d %q", rec.Code, rec.Body.String())
	}
}

func TestETagChangesWithDataset(t *testing.T) {
	a := etagOf("stations", time.Unix(1, 0).String(), "42")
	b := etagOf("stations", time.Unix(2, 0).String(), "42")
	if a == b {
		t.Fatal("etag should change when the dataset changes")
	}
}
