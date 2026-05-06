CREATE TABLE IF NOT EXISTS ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES affiliate_links(id) ON DELETE CASCADE,
    variant_a_post_id UUID REFERENCES posts(id),
    variant_b_post_id UUID REFERENCES posts(id),
    winner VARCHAR(1), -- 'a', 'b', or null (undecided)
    status VARCHAR(20) DEFAULT 'running', -- running, completed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
