-- Model modality: text, image or video.
ALTER TABLE models ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'text';
