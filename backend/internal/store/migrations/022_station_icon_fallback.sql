-- Existing databases: stations created before the favicon fallback have no
-- icon at all. Point them at the favicon proxy so they get a real mark.
-- Uploaded icons (/icons/...) and bundled logos (/logos/...) are left alone.
UPDATE stations
SET icon_url = '/api/v1/favicon/' || slug
WHERE icon_url IS NULL OR icon_url = '';
