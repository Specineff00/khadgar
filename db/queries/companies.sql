-- name: InsertCompany :exec
INSERT INTO companies(name, short_description, size, url_safe_name)
VALUES (
  sqlc.arg('name'),
  sqlc.arg('short_description'),
  sqlc.arg('size'),
  sqlc.arg('url_safe_name')
)
ON CONFLICT (name) DO NOTHING;

-- name: UpdateCompanyJobSite :exec
UPDATE companies
SET 
  working_url = sqlc.narg('working_url'),
  site_name = sqlc.narg('site_name'),
  last_checked_at = NOW(),
  attempts = attempts + 1,
  updated_at = NOW()
WHERE name = sqlc.arg('name');
  
