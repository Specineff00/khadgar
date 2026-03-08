-- name: UpsertWTTJScrapeMetaData :exec
INSERT INTO wttj_scrape_meta_data(id, next_page, completed, updated_at)
VALUES (
  1,
  sqlc.arg('next_page'),
  sqlc.arg('completed'),
  NOW()
)
ON CONFLICT (id) DO UPDATE SET
  next_page = EXCLUDED.next_page,
  completed = EXCLUDED.completed,
  updated_at = NOW();

-- name: GetWTTJScrapeMetaData :one
SELECT next_page, completed, updated_at
FROM wttj_scrape_meta_data
WHERE id = 1;


