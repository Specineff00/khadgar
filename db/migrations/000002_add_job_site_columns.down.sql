ALTER TABLE companies
  DROP COLUMN IF EXISTS working_url,
  DROP COLUMN IF EXISTS site_name,
  DROP COLUMN IF EXISTS last_checked_at,
  DROP COLUMN IF EXISTS attempts;
