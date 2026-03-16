ALTER TABLE IF EXISTS companies
  ADD COLUMN all_sites_checked BOOLEAN NOT NULL DEFAULT FALSE,
  DROP COLUMN IF EXISTS greenhouse_checked,
  DROP COLUMN IF EXISTS team_tailor_checked,
  DROP COLUMN IF EXISTS lever_checked,
  DROP COLUMN IF EXISTS workable_checked; 
