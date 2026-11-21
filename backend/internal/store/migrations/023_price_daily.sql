-- Daily price snapshots per model and station. One row per day, model and
-- station, written by the API's snapshot job; the model page reads aggregated
-- series from here instead of scanning the raw price_history table.
CREATE TABLE IF NOT EXISTS price_daily (
    day            DATE NOT NULL,
    model_id       TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    station_slug   TEXT NOT NULL,
    min_input_cny  NUMERIC(10, 4) NOT NULL,
    min_output_cny NUMERIC(10, 4) NOT NULL,
    offers         INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (day, model_id, station_slug)
);

CREATE INDEX IF NOT EXISTS price_daily_model_day_idx ON price_daily (model_id, day DESC);
CREATE INDEX IF NOT EXISTS price_daily_station_day_idx ON price_daily (station_slug, day DESC);
