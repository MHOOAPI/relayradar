package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SyncDemoStations tops up the demo dataset with stations that the current seed
// snapshot contains but the database does not. Existing rows are never touched,
// so live probe history, admin edits and uploaded icons survive a deploy that
// adds stations to the seed.
//
// The snapshot is deterministic and append-only: new stations always get IDs
// above the current maximum, which is what makes this safe without diffing
// every column.
func (s *Store) SyncDemoStations(ctx context.Context) error {
	snap, _ := BuildSeedSnapshot()
	if len(snap.Stations) == 0 {
		return nil
	}

	var maxStation, maxGroup int64
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM stations`).Scan(&maxStation); err != nil {
		return fmt.Errorf("max station id: %w", err)
	}
	if maxStation >= int64(len(snap.Stations)) {
		return nil
	}
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM station_groups`).Scan(&maxGroup); err != nil {
		return fmt.Errorf("max group id: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	inserted := map[int64]bool{}
	for _, st := range snap.Stations {
		if st.ID <= maxStation {
			continue
		}
		ct, err := tx.Exec(ctx, `
			INSERT INTO stations (id, slug, name, domain, base_url, tagline, rate_cny, min_topup_cny,
			                      payments, invoice, refund_policy, claimed, icon_url, listed_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT DO NOTHING`,
			st.ID, st.Slug, st.Name, st.Domain, st.BaseURL, st.Tagline, st.RateCNY, st.MinTopupCNY,
			st.Payments, st.Invoice, st.RefundPolicy, st.Claimed, st.IconURL, st.ListedAt)
		if err != nil {
			return fmt.Errorf("top up station %s: %w", st.Slug, err)
		}
		if ct.RowsAffected() > 0 {
			inserted[st.ID] = true
		}
	}

	newGroups := map[int64]bool{}
	for _, g := range snap.Groups {
		if g.ID <= maxGroup {
			continue
		}
		// A station can be skipped by ON CONFLICT (slug/domain already owned by
		// another row); its groups must be skipped too or the FK insert fails.
		if !inserted[g.StationID] {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO station_groups (id, station_id, slug, name, service, multiplier)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
			g.ID, g.StationID, g.Slug, g.Name, g.Service, g.Multiplier); err != nil {
			return fmt.Errorf("top up group %d: %w", g.ID, err)
		}
		newGroups[g.ID] = true
	}

	gmRows := make([][]any, 0, 1024)
	for _, gm := range snap.GroupModels {
		if !newGroups[gm.GroupID] {
			continue
		}
		gmRows = append(gmRows, []any{gm.GroupID, gm.ModelID, gm.InputCNY, gm.OutputCNY, gm.CacheReadCNY})
	}
	if len(gmRows) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"group_models"},
			[]string{"group_id", "model_id", "input_cny", "output_cny", "cache_read_cny"},
			pgx.CopyFromRows(gmRows)); err != nil {
			return fmt.Errorf("top up prices: %w", err)
		}
	}

	probeRows := make([][]any, 0, 4096)
	for _, p := range snap.Probes {
		if !newGroups[p.GroupID] {
			continue
		}
		probeRows = append(probeRows, []any{p.GroupID, p.Status, p.LatencyMS, p.ProbedAt})
	}
	if len(probeRows) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"probes"},
			[]string{"group_id", "status", "latency_ms", "probed_at"},
			pgx.CopyFromRows(probeRows)); err != nil {
			return fmt.Errorf("top up probes: %w", err)
		}
	}

	purityRows := make([][]any, 0, 256)
	for _, p := range snap.Purity {
		if !newGroups[p.GroupID] {
			continue
		}
		purityRows = append(purityRows, []any{p.GroupID, p.Score, p.Tier, p.DetectedModel, p.CheckedAt})
	}
	if len(purityRows) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"purity_checks"},
			[]string{"group_id", "score", "tier", "detected_model", "checked_at"},
			pgx.CopyFromRows(purityRows)); err != nil {
			return fmt.Errorf("top up purity: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `SELECT setval('stations_id_seq', (SELECT max(id) FROM stations))`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT setval('station_groups_id_seq', (SELECT max(id) FROM station_groups))`); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
