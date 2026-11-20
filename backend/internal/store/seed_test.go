package store

import "testing"

func TestBuildSeedSnapshot(t *testing.T) {
	snap, reviews := BuildSeedSnapshot()

	wantStations := 27 + len(loadExtraStations())
	if len(snap.Stations) != wantStations {
		t.Fatalf("stations = %d, want %d", len(snap.Stations), wantStations)
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

// Imported directory stations must be complete and unique; the seed assigns
// their IDs by order, so duplicates would corrupt the top-up path.
func TestImportedStations(t *testing.T) {
	extra := loadExtraStations()
	if len(extra) < 100 {
		t.Fatalf("imported stations = %d, want >= 100", len(extra))
	}
	slugs := map[string]bool{}
	domains := map[string]bool{}
	for _, st := range extra {
		if st.Slug == "" || st.Name == "" || st.Domain == "" || st.Tagline == "" {
			t.Errorf("incomplete station %+v", st)
		}
		if slugs[st.Slug] {
			t.Errorf("duplicate slug %s", st.Slug)
		}
		if domains[st.Domain] {
			t.Errorf("duplicate domain %s", st.Domain)
		}
		slugs[st.Slug], domains[st.Domain] = true, true
		if len(st.Services) == 0 {
			t.Errorf("station %s has no services", st.Slug)
		}
	}
}

// Every seeded station must carry an icon: a bundled logo when we have one,
// otherwise the favicon proxy path the API serves at runtime.
func TestSeedStationsHaveIcon(t *testing.T) {
	snap, _ := BuildSeedSnapshot()
	for _, st := range snap.Stations {
		if st.IconURL == "" {
			t.Errorf("station %s has no icon", st.Slug)
			continue
		}
		if bundled := seedLogos[st.Slug]; bundled == "" {
			want := "/api/v1/favicon/" + st.Slug
			if st.IconURL != want {
				t.Errorf("station %s icon = %q, want %q", st.Slug, st.IconURL, want)
			}
		} else if st.IconURL != bundled {
			t.Errorf("station %s icon = %q, want bundled logo %q", st.Slug, st.IconURL, bundled)
		}
	}
}
