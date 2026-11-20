package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Seed writes the deterministic demo dataset into PostgreSQL.
func (s *Store) Seed(ctx context.Context) error {
	snap, reviews := BuildSeedSnapshot()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, m := range snap.Models {
		if _, err := tx.Exec(ctx, `
			INSERT INTO models (id, vendor, vendor_name, name, service, context_window, input_usd, output_usd,
			                    released_at, coding_score, reasoning_score, free_tier, free_note, summary)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (id) DO NOTHING`,
			m.ID, m.Vendor, m.VendorName, m.Name, m.Service, m.ContextWindow, m.InputUSD, m.OutputUSD,
			m.ReleasedAt, m.CodingScore, m.ReasoningScore, m.FreeTier, m.FreeNote, m.Summary); err != nil {
			return err
		}
	}

	for _, st := range snap.Stations {
		if _, err := tx.Exec(ctx, `
			INSERT INTO stations (id, slug, name, domain, base_url, tagline, rate_cny, min_topup_cny,
			                      payments, invoice, refund_policy, claimed, icon_url, listed_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			st.ID, st.Slug, st.Name, st.Domain, st.BaseURL, st.Tagline, st.RateCNY, st.MinTopupCNY,
			st.Payments, st.Invoice, st.RefundPolicy, st.Claimed, st.IconURL, st.ListedAt); err != nil {
			return fmt.Errorf("seed station %s: %w", st.Slug, err)
		}
	}

	for _, g := range snap.Groups {
		if _, err := tx.Exec(ctx, `
			INSERT INTO station_groups (id, station_id, slug, name, service, multiplier)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			g.ID, g.StationID, g.Slug, g.Name, g.Service, g.Multiplier); err != nil {
			return err
		}
	}

	gmRows := make([][]any, 0, len(snap.GroupModels))
	for _, gm := range snap.GroupModels {
		gmRows = append(gmRows, []any{gm.GroupID, gm.ModelID, gm.InputCNY, gm.OutputCNY, gm.CacheReadCNY})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"group_models"},
		[]string{"group_id", "model_id", "input_cny", "output_cny", "cache_read_cny"},
		pgx.CopyFromRows(gmRows)); err != nil {
		return err
	}

	probeRows := make([][]any, 0, len(snap.Probes))
	for _, p := range snap.Probes {
		probeRows = append(probeRows, []any{p.GroupID, p.Status, p.LatencyMS, p.ProbedAt})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"probes"},
		[]string{"group_id", "status", "latency_ms", "probed_at"}, pgx.CopyFromRows(probeRows)); err != nil {
		return err
	}

	purityRows := make([][]any, 0, len(snap.Purity))
	for _, p := range snap.Purity {
		purityRows = append(purityRows, []any{p.GroupID, p.Score, p.Tier, p.DetectedModel, p.CheckedAt})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"purity_checks"},
		[]string{"group_id", "score", "tier", "detected_model", "checked_at"}, pgx.CopyFromRows(purityRows)); err != nil {
		return err
	}

	for _, r := range reviews {
		if _, err := tx.Exec(ctx, `
			INSERT INTO reviews (station_id, author, rating, body, verified, created_at)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			r.StationID, r.Author, r.Rating, r.Body, r.Verified, r.CreatedAt); err != nil {
			return err
		}
	}

	if len(snap.Prices) > 0 {
		priceRows := make([][]any, 0, len(snap.Prices))
		for _, p := range snap.Prices {
			priceRows = append(priceRows, []any{p.GroupID, p.ModelID, p.InputCNY, p.OutputCNY, p.RecordedAt})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"price_history"},
			[]string{"group_id", "model_id", "input_cny", "output_cny", "recorded_at"},
			pgx.CopyFromRows(priceRows)); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `SELECT setval('stations_id_seq', (SELECT max(id) FROM stations))`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT setval('station_groups_id_seq', (SELECT max(id) FROM station_groups))`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
