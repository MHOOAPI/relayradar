-- Webhook alert subscriptions (persisted so restarts keep them).
CREATE TABLE IF NOT EXISTS subscriptions (
    id           TEXT PRIMARY KEY,
    url          TEXT NOT NULL,
    secret       TEXT NOT NULL,
    kinds        TEXT[] NOT NULL DEFAULT '{}',
    stations     TEXT[] NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS subscriptions_last_sent_idx ON subscriptions (last_sent_at);
