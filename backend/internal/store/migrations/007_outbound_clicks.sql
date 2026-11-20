-- Outbound click counter for station guidance pages.
CREATE TABLE IF NOT EXISTS outbound_clicks (
    id           BIGSERIAL PRIMARY KEY,
    station_slug TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS outbound_clicks_slug_idx ON outbound_clicks (station_slug, created_at DESC);
