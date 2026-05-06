-- 003_link_health.up.sql
ALTER TABLE affiliate_links ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ;
ALTER TABLE affiliate_links ADD COLUMN IF NOT EXISTS health_status VARCHAR(20) DEFAULT 'unknown';
