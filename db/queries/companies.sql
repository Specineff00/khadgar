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
  updated_at = NOW(),
  should_retry = sqlc.arg('should_retry'),
  greenhouse_checked = sqlc.arg('greenhouse_checked'), 
  team_tailor_checked = sqlc.arg('team_tailor_checked'), 
  lever_checked = sqlc.arg('lever_checked'),
  workable_checked = sqlc.arg('workable_checked')
WHERE name = sqlc.arg('name');
  
-- name: GetUncheckedCompanies :many
SELECT name, url_safe_name FROM companies
  WHERE all_sites_checked is FALSE OR should_retry IS TRUE
  ORDER BY attempts ASC, id ASC;
