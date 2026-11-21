package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryPriceDailySnapshotIsIdempotent(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	day := time.Now().UTC()

	rows, err := m.SnapshotPriceDaily(ctx, day)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if rows == 0 {
		t.Fatal("snapshot wrote no rows")
	}

	snap, err := m.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot read: %v", err)
	}
	modelID := ""
	for _, gm := range snap.GroupModels {
		if gm.InputCNY != nil && *gm.InputCNY > 0 {
			modelID = gm.ModelID
			break
		}
	}
	if modelID == "" {
		t.Fatal("seed has no priced group models")
	}

	first, err := m.PriceDailyHistory(ctx, modelID, 90)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected one day, got %d", len(first))
	}
	if first[0].MinInputCNY <= 0 || first[0].Stations == 0 {
		t.Fatalf("bad aggregate: %+v", first[0])
	}

	if _, err := m.SnapshotPriceDaily(ctx, day); err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	second, err := m.PriceDailyHistory(ctx, modelID, 90)
	if err != nil {
		t.Fatalf("second history: %v", err)
	}
	if len(second) != 1 || second[0].Offers != first[0].Offers || second[0].Stations != first[0].Stations {
		t.Fatalf("snapshot is not idempotent: first=%+v second=%+v", first[0], second[0])
	}
}

func TestMemoryStationPriceHistory(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	day := time.Now().UTC()
	if _, err := m.SnapshotPriceDaily(ctx, day); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	snap, err := m.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot read: %v", err)
	}
	stationSlug := map[int64]string{}
	for _, st := range snap.Stations {
		stationSlug[st.ID] = st.Slug
	}
	groupStation := map[int64]int64{}
	for _, g := range snap.Groups {
		groupStation[g.ID] = g.StationID
	}
	var slug, modelID string
	for _, gm := range snap.GroupModels {
		if gm.InputCNY != nil && *gm.InputCNY > 0 {
			slug = stationSlug[groupStation[gm.GroupID]]
			modelID = gm.ModelID
			break
		}
	}
	if slug == "" || modelID == "" {
		t.Fatal("no priced group model found")
	}
	pts, err := m.StationPriceHistory(ctx, slug, modelID, 90)
	if err != nil {
		t.Fatalf("station history: %v", err)
	}
	if len(pts) != 1 || pts[0].MinInputCNY <= 0 || pts[0].Stations != 1 {
		t.Fatalf("unexpected station history: %+v", pts)
	}
}
