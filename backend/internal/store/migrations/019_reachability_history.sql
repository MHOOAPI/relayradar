-- Reachability history (credential-free homepage checks).
CREATE TABLE IF NOT EXISTS station_reachability_history (
    id           BIGSERIAL PRIMARY KEY,
    station_slug TEXT NOT NULL,
    status       TEXT NOT NULL,
    latency_ms   INT NOT NULL DEFAULT 0,
    checked_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS station_reachability_history_idx
    ON station_reachability_history (station_slug, checked_at DESC);
