-- Latest reachability probe per station (no credentials, homepage only).
CREATE TABLE IF NOT EXISTS station_reachability (
    station_slug TEXT PRIMARY KEY,
    status       TEXT NOT NULL,
    latency_ms   INT NOT NULL DEFAULT 0,
    checked_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
