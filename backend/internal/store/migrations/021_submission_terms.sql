-- Structured commercial terms captured with a station submission.
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS payments TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS invoice BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS refund_policy TEXT NOT NULL DEFAULT '';
