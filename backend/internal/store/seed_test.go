package store

import "testing"

func TestBuildSeedSnapshot(t *testing.T) {
	snap, reviews := BuildSeedSnapshot()

	if len(snap.Stations) != 26 {
		t.Fatalf("stations = %d, want 26", len(snap.Stations))
	}
	if len(snap.Models) < 100 {
		t.Fatalf("models = %d, want at least 100", len(snap.Models))
	}
	if len(snap.Groups) < 30 {
		t.Fatalf("groups = %d, want >= 30", len(snap.Groups))
	}
	if len(snap.Probes) < 5000 {
		t.Fatalf("probes = %d, want >= 5000", len(snap.Probes))
	}
	if len(snap.GroupModels) == 0 {
		t.Fatal("no model prices generated")
	}
	if len(reviews) == 0 {
		t.Fatal("no reviews generated")
	}

	groupService := map[int64]string{}
	for _, g := range snap.Groups {
		groupService[g.ID] = g.Service
	}
	for _, p := range snap.Purity {
		if groupService[p.GroupID] != "claude" {
			t.Fatalf("purity measured on non-claude group %d (%s)", p.GroupID, groupService[p.GroupID])
		}
	}
	if len(snap.Purity) == 0 {
		t.Fatal("no purity checks generated")
	}

	for _, gm := range snap.GroupModels {
		if gm.InputCNY == nil || *gm.InputCNY <= 0 {
			t.Fatalf("model %s has non-positive input price", gm.ModelID)
		}
		if gm.OutputCNY == nil || *gm.OutputCNY <= 0 {
			t.Fatalf("model %s has non-positive output price", gm.ModelID)
		}
	}

	ids := map[int64]bool{}
	for _, st := range snap.Stations {
		if ids[st.ID] {
			t.Fatalf("duplicate station id %d", st.ID)
		}
		ids[st.ID] = true
	}
	for _, r := range reviews {
		if !ids[r.StationID] {
			t.Fatalf("review points at unknown station %d", r.StationID)
		}
	}
}
