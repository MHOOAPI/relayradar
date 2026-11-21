-- Price alert thresholds. 0 means "no threshold": any price move fires the
-- price_change kind; min_delta_pct requires a move of at least that percent,
-- max_price_cny only delivers when the new price is at or below the ceiling.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS min_delta_pct NUMERIC(6, 2) NOT NULL DEFAULT 0;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS max_price_cny NUMERIC(10, 4) NOT NULL DEFAULT 0;
