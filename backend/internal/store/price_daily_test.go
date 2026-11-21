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

func TestMemoryPriceDrops(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	day1 := time.Now().UTC().AddDate(0, 0, -1)
	day2 := time.Now().UTC()

	// Make yesterday's price 25% higher for one model, then restore it today.
	idx := -1
	for i, gm := range m.snap.GroupModels {
		if gm.InputCNY != nil && *gm.InputCNY > 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("no priced group model")
	}
	orig := *m.snap.GroupModels[idx].InputCNY
	higher := orig * 1.25
	m.snap.GroupModels[idx].InputCNY = &higher
	if _, err := m.SnapshotPriceDaily(ctx, day1); err != nil {
		t.Fatalf("day1 snapshot: %v", err)
	}
	m.snap.GroupModels[idx].InputCNY = &orig
	if _, err := m.SnapshotPriceDaily(ctx, day2); err != nil {
		t.Fatalf("day2 snapshot: %v", err)
	}

	drops, err := m.PriceDrops(ctx, 30)
	if err != nil {
		t.Fatalf("drops: %v", err)
	}
	if len(drops) == 0 {
		t.Fatal("expected at least one drop")
	}
	found := false
	for _, d := range drops {
		if d.ModelID == m.snap.GroupModels[idx].ModelID && d.NewPriceCNY < d.OldPriceCNY {
			found = true
		}
	}
	if !found {
		t.Fatalf("drop for the modified model missing: %+v", drops)
	}
}

func TestMemorySnapshotRecordsPriceHistory(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	idx := -1
	for i, gm := range m.snap.GroupModels {
		if gm.InputCNY != nil && *gm.InputCNY > 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("no priced group model")
	}
	if _, err := m.SnapshotPriceDaily(ctx, time.Now().UTC().AddDate(0, 0, -1)); err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	before := len(m.snap.Prices)

	lower := *m.snap.GroupModels[idx].InputCNY * 0.8
	m.snap.GroupModels[idx].InputCNY = &lower
	if _, err := m.SnapshotPriceDaily(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if len(m.snap.Prices) != before+1 {
		t.Fatalf("expected exactly one new history point, got %d -> %d", before, len(m.snap.Prices))
	}
	last := m.snap.Prices[len(m.snap.Prices)-1]
	if last.ModelID != m.snap.GroupModels[idx].ModelID || last.InputCNY == nil || *last.InputCNY != lower {
		t.Fatalf("history point does not record the new price: %+v", last)
	}
}
