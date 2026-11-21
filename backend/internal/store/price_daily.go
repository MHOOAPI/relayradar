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
