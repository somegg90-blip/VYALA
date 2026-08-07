-- Accounts, Users, Account_Users, Repositories (Same as before)
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_org_id BIGINT UNIQUE,
    name VARCHAR(255) NOT NULL,
    plan_tier VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_user_id BIGINT UNIQUE NOT NULL,
    email VARCHAR(255),
    name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS account_users (
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'admin',
    PRIMARY KEY (account_id, user_id)
);

CREATE TABLE IF NOT EXISTS repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    github_repo_id BIGINT UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    is_private BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Scans (Metadata + Raw JSON)
CREATE TABLE IF NOT EXISTS scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    commit_sha VARCHAR(40) NOT NULL,
    branch VARCHAR(255) NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_cbom JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_scans_repo_created ON scans(repo_id, created_at DESC);

-- Findings (Lifecycle Tracking - The improved version)
CREATE TABLE IF NOT EXISTS findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    finding_id VARCHAR(64) NOT NULL,
    type VARCHAR(50),
    file TEXT NOT NULL, -- Changed from VARCHAR(512) to TEXT to support deep monorepo paths
    line INTEGER NOT NULL,
    algorithm VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    category VARCHAR(100) NOT NULL,
    exposure_estimate TEXT,
    suggested_replacement TEXT,
    rule_id VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- 'open', 'resolved', 'suppressed'
    first_seen_scan_id UUID REFERENCES scans(id),
    last_seen_scan_id UUID REFERENCES scans(id),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    UNIQUE (repo_id, finding_id)
);
CREATE INDEX IF NOT EXISTS idx_findings_repo_status ON findings(repo_id, status);

-- Wishlist & Early Access Passes
CREATE TABLE IF NOT EXISTS wishlist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    email VARCHAR(255) NOT NULL,
    role VARCHAR(100),
    pass_code VARCHAR(64) UNIQUE,
    wish TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wishlist_email ON wishlist(email);

-- Trigger to automatically update the updated_at column on row modifications
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scans_updated_at BEFORE UPDATE ON scans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_findings_updated_at BEFORE UPDATE ON findings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();