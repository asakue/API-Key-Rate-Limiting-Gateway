-- Enable UUID extension if supported
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'admin',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tiers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(64) UNIQUE NOT NULL,
    description TEXT,
    rate_limit_per_min INT NOT NULL DEFAULT 60,
    rate_limit_per_sec INT NOT NULL DEFAULT 10,
    monthly_quota INT NOT NULL DEFAULT 10000,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier_id VARCHAR(64) NOT NULL REFERENCES tiers(id),
    name VARCHAR(128) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash VARCHAR(128) UNIQUE NOT NULL,
    scopes JSONB NOT NULL DEFAULT '["*"]'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- active, revoked, expired
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

CREATE TABLE IF NOT EXISTS routes (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    path_prefix VARCHAR(255) NOT NULL,
    upstream_url VARCHAR(512) NOT NULL,
    strip_prefix BOOLEAN NOT NULL DEFAULT FALSE,
    timeout_ms INT NOT NULL DEFAULT 5000,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routes_path ON routes(path_prefix);

CREATE TABLE IF NOT EXISTS request_aggregates (
    id BIGSERIAL PRIMARY KEY,
    bucket_time TIMESTAMP WITH TIME ZONE NOT NULL,
    key_id VARCHAR(64),
    route_id VARCHAR(64),
    status_bucket VARCHAR(16) NOT NULL, -- 2xx, 4xx, 5xx, 429
    status_code INT NOT NULL,
    request_count INT NOT NULL DEFAULT 1,
    total_latency_ms BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_req_agg_bucket ON request_aggregates(bucket_time);
CREATE INDEX IF NOT EXISTS idx_req_agg_key ON request_aggregates(key_id);

-- Default Tiers
INSERT INTO tiers (id, name, description, rate_limit_per_min, rate_limit_per_sec, monthly_quota)
VALUES 
    ('tier_free', 'Free', 'Free tier for developers and prototyping', 60, 5, 10000),
    ('tier_starter', 'Starter', 'Starter tier for growth and small applications', 300, 20, 100000),
    ('tier_pro', 'Pro', 'Production tier with high RPS and generous quota', 1200, 100, 1000000)
ON CONFLICT (name) DO NOTHING;

-- Default Mock Upstream Route
INSERT INTO routes (id, name, path_prefix, upstream_url, strip_prefix, timeout_ms, is_active)
VALUES
    ('route_mock', 'Mock Backend Upstream', '/mock', 'http://mock-backend:8080', true, 5000, true)
ON CONFLICT (id) DO NOTHING;

