-- Reviewer note for claim edit decisions.
ALTER TABLE claim_edits ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '';
