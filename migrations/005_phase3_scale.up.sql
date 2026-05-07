-- 005_phase3_scale.up.sql
-- Phase 3: Scale - Self-learning AI, Circuit Breaker, Optimized Timing, Weekly Reports

-- Persona weights: tracks performance of each persona per user
CREATE TABLE IF NOT EXISTS persona_weights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    persona VARCHAR(50) NOT NULL,
    weight DECIMAL(5,3) DEFAULT 1.000, -- 0.000 to 5.000, higher = more likely to be selected
    total_posts INT DEFAULT 0,
    total_clicks INT DEFAULT 0,
    avg_engagement DECIMAL(8,2) DEFAULT 0.00,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, persona)
);

-- Format weights: tracks performance of each format per user
CREATE TABLE IF NOT EXISTS format_weights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    format VARCHAR(30) NOT NULL,
    weight DECIMAL(5,3) DEFAULT 1.000,
    total_posts INT DEFAULT 0,
    total_clicks INT DEFAULT 0,
    avg_engagement DECIMAL(8,2) DEFAULT 0.00,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, format)
);

-- Posting time weights: tracks best hours for posting per user
CREATE TABLE IF NOT EXISTS time_weights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hour_wib INT NOT NULL CHECK (hour_wib >= 0 AND hour_wib <= 23),
    weight DECIMAL(5,3) DEFAULT 1.000,
    total_posts INT DEFAULT 0,
    total_clicks INT DEFAULT 0,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, hour_wib)
);

-- Weekly reports: stores generated insight reports
CREATE TABLE IF NOT EXISTS weekly_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    total_posts INT DEFAULT 0,
    total_clicks INT DEFAULT 0,
    total_views INT DEFAULT 0,
    best_persona VARCHAR(50),
    best_format VARCHAR(30),
    best_hour INT,
    top_post_id UUID REFERENCES posts(id) ON DELETE SET NULL,
    recommendations JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, week_start)
);

-- Circuit breaker events: enhanced with severity and cooldown tracking
ALTER TABLE circuit_breaker ADD COLUMN IF NOT EXISTS severity VARCHAR(20) DEFAULT 'warning';
ALTER TABLE circuit_breaker ADD COLUMN IF NOT EXISTS cooldown_until TIMESTAMPTZ;
ALTER TABLE circuit_breaker ADD COLUMN IF NOT EXISTS auto_resolved BOOLEAN DEFAULT false;
ALTER TABLE circuit_breaker ADD COLUMN IF NOT EXISTS notes TEXT;

-- Add daily_post_count to threads_accounts for rate limiting
ALTER TABLE threads_accounts ADD COLUMN IF NOT EXISTS daily_post_count INT DEFAULT 0;
ALTER TABLE threads_accounts ADD COLUMN IF NOT EXISTS last_post_reset_at DATE DEFAULT CURRENT_DATE;
ALTER TABLE threads_accounts ADD COLUMN IF NOT EXISTS max_daily_posts INT DEFAULT 25;
ALTER TABLE threads_accounts ADD COLUMN IF NOT EXISTS flagged_count INT DEFAULT 0;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_persona_weights_user ON persona_weights(user_id);
CREATE INDEX IF NOT EXISTS idx_format_weights_user ON format_weights(user_id);
CREATE INDEX IF NOT EXISTS idx_time_weights_user ON time_weights(user_id);
CREATE INDEX IF NOT EXISTS idx_weekly_reports_user_week ON weekly_reports(user_id, week_start DESC);
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_account_resolved ON circuit_breaker(account_id, resolved_at);
CREATE INDEX IF NOT EXISTS idx_posts_persona_published ON posts(persona, published_at) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_posts_format_published ON posts(format, published_at) WHERE status = 'published';
