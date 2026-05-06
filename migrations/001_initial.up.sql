-- 001_initial.up.sql
-- Threads Affiliate Automation Platform - Initial Schema
-- PostgreSQL 16

-- 1. Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    plan VARCHAR(20) DEFAULT 'trial',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Threads Accounts
CREATE TABLE IF NOT EXISTS threads_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    threads_user_id VARCHAR(100),
    access_token TEXT,
    refresh_token TEXT,
    persona VARCHAR(50),
    niche VARCHAR(50),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Products
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(500),
    price DECIMAL(12,2),
    category VARCHAR(100),
    platform VARCHAR(20) NOT NULL,
    image_url TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Affiliate Links
CREATE TABLE IF NOT EXISTS affiliate_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    original_url TEXT NOT NULL,
    short_slug VARCHAR(20) UNIQUE NOT NULL,
    platform VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    click_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 5. Posts
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES threads_accounts(id) ON DELETE CASCADE,
    link_id UUID REFERENCES affiliate_links(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    link_placement VARCHAR(30),
    persona VARCHAR(50),
    format VARCHAR(20),
    scheduled_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    thread_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. Post Analytics
CREATE TABLE IF NOT EXISTS post_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    views INT DEFAULT 0,
    likes INT DEFAULT 0,
    replies INT DEFAULT 0,
    reposts INT DEFAULT 0,
    clicks INT DEFAULT 0,
    fetched_at TIMESTAMPTZ DEFAULT NOW()
);

-- 7. Click Logs
CREATE TABLE IF NOT EXISTS click_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES affiliate_links(id) ON DELETE CASCADE,
    hashed_ip VARCHAR(64),
    user_agent TEXT,
    referrer TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 8. Circuit Breaker
CREATE TABLE IF NOT EXISTS circuit_breaker (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES threads_accounts(id) ON DELETE CASCADE,
    event_type VARCHAR(50),
    triggered_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_affiliate_links_short_slug ON affiliate_links(short_slug);
CREATE INDEX IF NOT EXISTS idx_posts_status_scheduled_at ON posts(status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_posts_account_id ON posts(account_id);
CREATE INDEX IF NOT EXISTS idx_click_logs_link_id_created_at ON click_logs(link_id, created_at);
