-- Optional provider filter for provider_incident subscriptions.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS providers TEXT[] NOT NULL DEFAULT '{}';
