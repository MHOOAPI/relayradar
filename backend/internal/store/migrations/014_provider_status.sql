-- Upstream model provider status snapshots.
CREATE TABLE IF NOT EXISTS provider_status (
    id         BIGSERIAL PRIMARY KEY,
    provider   TEXT NOT NULL,
    status     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS provider_status_idx ON provider_status (provider, checked_at DESC);
