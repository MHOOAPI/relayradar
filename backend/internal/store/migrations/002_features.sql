-- Feature additions: station icons, model free tiers, submissions and claims.

ALTER TABLE models ADD COLUMN IF NOT EXISTS free_tier BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE models ADD COLUMN IF NOT EXISTS free_note TEXT NOT NULL DEFAULT '';
ALTER TABLE stations ADD COLUMN IF NOT EXISTS icon_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS submissions (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    domain      TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    contact     TEXT NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS claims (
    id          BIGSERIAL PRIMARY KEY,
    station_id  BIGINT NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    token       TEXT UNIQUE NOT NULL,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS claims_token_idx ON claims (token);
