-- name: InsertCompany :exec
INSERT INTO companies(name, url_safe_name, short_description, size)
VALUES (
  sqlc.arg('name'),
  regexp_replace(sqlc.arg('url_safe_name'), '-[0-9]+$', ''), -- Removes dash and number suffix
  sqlc.arg('short_description'),
  sqlc.arg('size')
)
ON CONFLICT (name) DO NOTHING;

-- name: UpdateWorkableJobSite :exec
UPDATE companies
SET 
  working_url = sqlc.narg('working_url'),
  site_name = sqlc.narg('site_name'),
  last_checked_at = NOW(),
  attempts = attempts + 1,
  updated_at = NOW(),
  should_retry = sqlc.arg('should_retry'),
  workable_checked = sqlc.arg('workable_checked')
WHERE name = sqlc.arg('name');

-- name: UpdateTeamTailorJobSite :exec
UPDATE companies
SET 
  working_url = sqlc.narg('working_url'),
  site_name = sqlc.narg('site_name'),
  last_checked_at = NOW(),
  attempts = attempts + 1,
  updated_at = NOW(),
  should_retry = sqlc.arg('should_retry'),
  team_tailor_checked = sqlc.arg('team_tailor_checked') 
WHERE name = sqlc.arg('name');

-- name: UpdateGreenhouseJobSite :exec
UPDATE companies
SET 
  working_url = sqlc.narg('working_url'),
  site_name = sqlc.narg('site_name'),
  last_checked_at = NOW(),
  attempts = attempts + 1,
  updated_at = NOW(),
  should_retry = sqlc.arg('should_retry'),
  greenhouse_checked = sqlc.arg('greenhouse_checked') 
WHERE name = sqlc.arg('name');

-- name: UpdateLeverJobSite :exec
UPDATE companies
SET 
  working_url = sqlc.narg('working_url'),
  site_name = sqlc.narg('site_name'),
  last_checked_at = NOW(),
  attempts = attempts + 1,
  updated_at = NOW(),
  should_retry = sqlc.arg('should_retry'),
  lever_checked = sqlc.arg('lever_checked')
WHERE name = sqlc.arg('name');

-- name: GetUncheckedCompanies :many
SELECT name, url_safe_name FROM companies
  WHERE greenhouse_checked is FALSE
  OR lever_checked is FALSE
  OR team_tailor_checked is FALSE
  OR should_retry IS TRUE
  ORDER BY attempts ASC, id ASC;

-- name: GetAllDiscoveredSitesBySiteName :many
SELECT url_safe_name, id FROM companies
  WHERE site_name = sqlc.arg(site_name)
  ORDER BY site_name ASC;

-- name: ResetCompaniesJobColumns :exec
UPDATE companies
SET
  working_url = NULL,
  site_name = NULL,
  last_checked_at = NULL,
  attempts = 0,
  should_retry = FALSE,
  workable_checked = FALSE,
  greenhouse_checked = FALSE,
  team_tailor_checked = FALSE,
  lever_checked = FALSE;
