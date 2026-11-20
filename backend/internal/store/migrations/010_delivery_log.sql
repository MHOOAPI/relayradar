-- Delivery log for alert subscriptions, with the payload kept for replay.
CREATE TABLE IF NOT EXISTS delivery_log (
    id              BIGSERIAL PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    event           TEXT NOT NULL DEFAULT 'change',
    status          INT NOT NULL DEFAULT 0,
    ok              BOOLEAN NOT NULL DEFAULT false,
    duration_ms     INT NOT NULL DEFAULT 0,
    response        TEXT NOT NULL DEFAULT '',
    payload         TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS delivery_log_sub_idx ON delivery_log (subscription_id, created_at DESC);
