-- RelayRadar initial schema.

CREATE TABLE IF NOT EXISTS stations (
    id              BIGSERIAL PRIMARY KEY,
    slug            TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    domain          TEXT NOT NULL,
    base_url        TEXT NOT NULL,
    tagline         TEXT NOT NULL DEFAULT '',
    rate_cny        NUMERIC(8, 3) NOT NULL DEFAULT 1.0,
    min_topup_cny   NUMERIC(10, 2) NOT NULL DEFAULT 0,
    payments        TEXT[] NOT NULL DEFAULT '{}',
    invoice         BOOLEAN NOT NULL DEFAULT FALSE,
    refund_policy   TEXT NOT NULL DEFAULT '',
    claimed         BOOLEAN NOT NULL DEFAULT FALSE,
    listed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS station_groups (
    id          BIGSERIAL PRIMARY KEY,
    station_id  BIGINT NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    service     TEXT NOT NULL,
    multiplier  NUMERIC(6, 3) NOT NULL DEFAULT 1.0,
    UNIQUE (station_id, slug)
);

CREATE TABLE IF NOT EXISTS models (
    id               TEXT PRIMARY KEY,
    vendor           TEXT NOT NULL,
    vendor_name      TEXT NOT NULL,
    name             TEXT NOT NULL,
    service          TEXT NOT NULL,
    context_window   INTEGER NOT NULL DEFAULT 0,
    input_usd        NUMERIC(10, 4) NOT NULL DEFAULT 0,
    output_usd       NUMERIC(10, 4) NOT NULL DEFAULT 0,
    released_at      DATE,
    coding_score     NUMERIC(4, 1) NOT NULL DEFAULT 0,
    reasoning_score  NUMERIC(4, 1) NOT NULL DEFAULT 0,
    summary          TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS group_models (
    group_id       BIGINT NOT NULL REFERENCES station_groups(id) ON DELETE CASCADE,
    model_id       TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    input_cny      NUMERIC(10, 4),
    output_cny     NUMERIC(10, 4),
    cache_read_cny NUMERIC(10, 4),
    PRIMARY KEY (group_id, model_id)
);

CREATE TABLE IF NOT EXISTS probes (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT NOT NULL REFERENCES station_groups(id) ON DELETE CASCADE,
    status      TEXT NOT NULL CHECK (status IN ('ok', 'slow', 'down')),
    latency_ms  INTEGER,
    probed_at   TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS probes_group_time_idx ON probes (group_id, probed_at DESC);

CREATE TABLE IF NOT EXISTS purity_checks (
    id              BIGSERIAL PRIMARY KEY,
    group_id        BIGINT NOT NULL REFERENCES station_groups(id) ON DELETE CASCADE,
    score           INTEGER NOT NULL,
    tier            TEXT NOT NULL,
    detected_model  TEXT,
    checked_at      TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS purity_group_time_idx ON purity_checks (group_id, checked_at DESC);

CREATE TABLE IF NOT EXISTS reviews (
    id          BIGSERIAL PRIMARY KEY,
    station_id  BIGINT NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    author      TEXT NOT NULL,
    rating      SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    body        TEXT NOT NULL,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_reports (
    id          TEXT PRIMARY KEY,
    target_url  TEXT NOT NULL,
    model       TEXT NOT NULL,
    passed      BOOLEAN NOT NULL,
    payload     JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_created_idx ON audit_reports (created_at DESC);
