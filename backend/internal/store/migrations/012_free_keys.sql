-- User-submitted free API keys. Keys are stored so they can be claimed; the
-- list endpoint only ever returns a masked form.
CREATE TABLE IF NOT EXISTS free_keys (
    id              BIGSERIAL PRIMARY KEY,
    provider        TEXT NOT NULL DEFAULT '',
    base_url        TEXT NOT NULL,
    api_key         TEXT NOT NULL,
    models          TEXT NOT NULL DEFAULT '',
    note            TEXT NOT NULL DEFAULT '',
    submitter       TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'unknown',
    claims          INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_checked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS free_keys_created_idx ON free_keys (created_at DESC);
