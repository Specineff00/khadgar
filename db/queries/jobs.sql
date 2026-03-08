-- name: UpsertJob :exec
INSERT INTO jobs (company_id, external_id, search_term, title, url, location, active, last_seen_at)
VALUES(
  sqlc.arg('company_id'),
  sqlc.arg('external_id'),
  sqlc.arg('search_term'),
  sqlc.arg('title'),
  sqlc.arg('url'),
  sqlc.arg('location'),
  TRUE,
  NOW()
)
ON CONFLICT (company_id, external_id) DO UPDATE SET
  search_term = EXCLUDED.search_term,
  title = EXCLUDED.title,
  url = EXCLUDED.url,
  location = EXCLUDED.location,
  active = true,
  last_seen_at = NOW();

-- name: DeactivateStaleJobs :exec
UPDATE jobs
SET active = FALSE
WHERE company_id = sqlc.arg('company_id')
  AND last_seen_at <
  sqlc.arg('scrape_started_at');
