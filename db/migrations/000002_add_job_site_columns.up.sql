ALTER TABLE companies
  ADD COLUMN working_url TEXT,
  ADD COLUMN site_name TEXT,
  ADD COLUMN last_checked_at TIMESTAMPTZ,
  ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;

