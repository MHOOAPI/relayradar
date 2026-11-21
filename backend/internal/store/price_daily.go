package store

import (
	"context"
	"time"
)

// PriceDaily is one model's aggregated price for a station on a given day.
type PriceDaily struct {
	Day          time.Time
	ModelID      string
	StationSlug  string
	MinInputCNY  float64
	MinOutputCNY float64
	Offers       int
}

// PriceDailyPoint is the per-day aggregate exposed by the API: the cheapest
// station price for the model that day.
type PriceDailyPoint struct {
	Day          time.Time `json:"day"`
	MinInputCNY  float64   `json:"minInputCny"`
	MinOutputCNY float64   `json:"minOutputCny"`
	Stations     int       `json:"stations"`
	Offers       int       `json:"offers"`
}

// SnapshotPriceDaily writes today's per-station minimums for every model and
// returns the number of rows touched. It is idempotent within a day.
func (s *Store) SnapshotPriceDaily(ctx context.Context, day time.Time) (int, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO price_daily (day, model_id, station_slug, min_input_cny, min_output_cny, offers)
		SELECT $1::date, gm.model_id, st.slug, MIN(gm.input_cny), MIN(gm.output_cny), COUNT(*)
		FROM group_models gm
		JOIN station_groups g ON g.id = gm.group_id
		JOIN stations st ON st.id = g.station_id
		WHERE gm.input_cny IS NOT NULL AND gm.input_cny > 0
		GROUP BY gm.model_id, st.slug
		ON CONFLICT (day, model_id, station_slug) DO UPDATE
		SET min_input_cny = EXCLUDED.min_input_cny,
		    min_output_cny = EXCLUDED.min_output_cny,
		    offers = EXCLUDED.offers`, day)
	if err != nil {
		return 0, err
	}
	// Record the first price of every group/model once, then only append when
	// the price actually changes. This feeds the existing price_change feed
	// (and therefore price alerts) with real observations.
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO price_history (group_id, model_id, input_cny, output_cny, recorded_at)
		SELECT gm.group_id, gm.model_id, gm.input_cny, gm.output_cny, now()
		FROM group_models gm
		WHERE gm.input_cny IS NOT NULL AND gm.input_cny > 0
		  AND (
		    SELECT ph.input_cny FROM price_history ph
		    WHERE ph.group_id = gm.group_id AND ph.model_id = gm.model_id
		    ORDER BY ph.recorded_at DESC
		    LIMIT 1
		  ) IS DISTINCT FROM gm.input_cny`); err != nil {
		return int(tag.RowsAffected()), err
	}
	return int(tag.RowsAffected()), nil
}

// PriceDailyHistory returns the daily cheapest price for a model, oldest first.
func (s *Store) PriceDailyHistory(ctx context.Context, modelID string, days int) ([]PriceDailyPoint, error) {
	if days <= 0 {
		days = 90
	}
	rows, err := s.pool.Query(ctx, `
		SELECT day, MIN(min_input_cny), MIN(min_output_cny),
		       COUNT(DISTINCT station_slug), SUM(offers)
		FROM price_daily
		WHERE model_id = $1 AND day > CURRENT_DATE - $2::int
		GROUP BY day
		ORDER BY day`, modelID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PriceDailyPoint, 0, days)
	for rows.Next() {
		var p PriceDailyPoint
		if err := rows.Scan(&p.Day, &p.MinInputCNY, &p.MinOutputCNY, &p.Stations, &p.Offers); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// StationPriceHistory returns one station's daily price for a single model.
func (s *Store) StationPriceHistory(ctx context.Context, slug, modelID string, days int) ([]PriceDailyPoint, error) {
	if days <= 0 {
		days = 90
	}
	rows, err := s.pool.Query(ctx, `
		SELECT day, min_input_cny, min_output_cny, 1, offers
		FROM price_daily
		WHERE station_slug = $1 AND model_id = $2 AND day > CURRENT_DATE - $3::int
		ORDER BY day`, slug, modelID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PriceDailyPoint, 0, days)
	for rows.Next() {
		var p PriceDailyPoint
		if err := rows.Scan(&p.Day, &p.MinInputCNY, &p.MinOutputCNY, &p.Stations, &p.Offers); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PriceDropRow is one station/model pair whose price fell inside the window.
type PriceDropRow struct {
	StationSlug string
	ModelID     string
	OldPriceCNY float64
	NewPriceCNY float64
	OldDay      time.Time
	NewDay      time.Time
}

// PriceDrops compares the first and last snapshot in the window for every
// station/model pair and returns the ones whose input price fell.
func (s *Store) PriceDrops(ctx context.Context, days int) ([]PriceDropRow, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := s.pool.Query(ctx, `
		WITH w AS (
			SELECT station_slug, model_id, day, min_input_cny,
			       FIRST_VALUE(min_input_cny) OVER (PARTITION BY station_slug, model_id ORDER BY day) AS first_price,
			       FIRST_VALUE(day) OVER (PARTITION BY station_slug, model_id ORDER BY day) AS first_day,
			       LAST_VALUE(min_input_cny) OVER (
			           PARTITION BY station_slug, model_id ORDER BY day
			           ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS last_price,
			       LAST_VALUE(day) OVER (
			           PARTITION BY station_slug, model_id ORDER BY day
			           ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS last_day
			FROM price_daily
			WHERE day > CURRENT_DATE - $1::int
		)
		SELECT DISTINCT ON (station_slug, model_id)
		       station_slug, model_id, first_price, last_price, first_day, last_day
		FROM w
		WHERE last_price < first_price * 0.99
		ORDER BY station_slug, model_id, last_day DESC`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PriceDropRow, 0, 32)
	for rows.Next() {
		var r PriceDropRow
		if err := rows.Scan(&r.StationSlug, &r.ModelID, &r.OldPriceCNY, &r.NewPriceCNY, &r.OldDay, &r.NewDay); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
