-- name: InsertCompany :exec
INSERT INTO companies(name, url_safe_name, short_description, size)
VALUES (
  sqlc.arg('name'),
  regexp_replace(sqlc.arg('url_safe_name'), '-[0-9]+$', ''), -- Removes dash and number suffix
  sqlc.arg('short_description'),
  sqlc.arg('size')
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
  

-- name: GetUncheckedCompanies :many
SELECT id, url_safe_name FROM companies
  WHERE site_name is NULL
  ORDER BY attempts ASC, id ASC;
