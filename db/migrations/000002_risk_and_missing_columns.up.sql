-- Add risk_reports table
CREATE TABLE IF NOT EXISTS risk_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL DEFAULT '',
    category VARCHAR(30) NOT NULL DEFAULT 'other',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    admin_note TEXT NOT NULL DEFAULT '',
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_reports_target_id ON risk_reports(target_id);
CREATE INDEX IF NOT EXISTS idx_risk_reports_client_id ON risk_reports(client_id);
CREATE INDEX IF NOT EXISTS idx_risk_reports_status ON risk_reports(status);
CREATE INDEX IF NOT EXISTS idx_risk_reports_created_at ON risk_reports(created_at DESC);

-- Add risk_list table (social account blacklist)
CREATE TABLE IF NOT EXISTS risk_list (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(30) NOT NULL,
    provider_uid VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL DEFAULT '',
    report_id UUID REFERENCES risk_reports(id) ON DELETE SET NULL,
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_uid)
);

CREATE INDEX IF NOT EXISTS idx_risk_list_provider ON risk_list(provider, provider_uid);
CREATE INDEX IF NOT EXISTS idx_risk_list_user_id ON risk_list(user_id) WHERE user_id IS NOT NULL;

-- Add missing columns to oidc_clients
ALTER TABLE oidc_clients ADD COLUMN IF NOT EXISTS client_secret_plain TEXT NOT NULL DEFAULT '';
ALTER TABLE oidc_clients ADD COLUMN IF NOT EXISTS require_email_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE oidc_clients ADD COLUMN IF NOT EXISTS is_confidential BOOLEAN NOT NULL DEFAULT TRUE;

-- Add subject column to oauth2_sessions if using unified table (for forward compatibility)
-- The initial schema already has subject in per-type tables, so this is a no-op guard.
-- If oauth2_sessions exists (custom setup), add subject there too.
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'oauth2_sessions') THEN
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'oauth2_sessions' AND column_name = 'subject') THEN
            ALTER TABLE oauth2_sessions ADD COLUMN subject VARCHAR(255) NOT NULL DEFAULT '';
            CREATE INDEX idx_oauth2_sessions_subject ON oauth2_sessions(subject);
        END IF;
    END IF;
END
$$;
