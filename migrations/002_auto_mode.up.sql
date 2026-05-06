ALTER TABLE threads_accounts ADD COLUMN IF NOT EXISTS auto_mode BOOLEAN DEFAULT false;
ALTER TABLE threads_accounts ADD COLUMN IF NOT EXISTS auto_mode_enabled_at TIMESTAMPTZ;
