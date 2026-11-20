-- Claimed-profile edits waiting for admin review.
CREATE TABLE IF NOT EXISTS claim_edits (
    id            BIGSERIAL PRIMARY KEY,
    station_id    BIGINT NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    tagline       TEXT NOT NULL DEFAULT '',
    refund_policy TEXT NOT NULL DEFAULT '',
    invoice       BOOLEAN NOT NULL DEFAULT false,
    min_topup_cny DOUBLE PRECISION NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS claim_edits_status_idx ON claim_edits (status, created_at DESC);
