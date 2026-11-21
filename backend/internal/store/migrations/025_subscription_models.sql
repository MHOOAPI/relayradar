-- Model-scoped subscriptions: an empty array means all models. When set, only
-- change events for those models are delivered.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS models TEXT[] NOT NULL DEFAULT '{}';
