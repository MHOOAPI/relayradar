-- Email channel for alert subscriptions.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'webhook';
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';
