-- Backfill station icons with the favicon proxy when no custom logo exists.
UPDATE stations
SET icon_url = '/api/v1/favicon/' || slug
WHERE icon_url IS NULL OR icon_url = '';
