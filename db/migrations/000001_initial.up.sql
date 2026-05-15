-- Initial schema for OIDC authentication platform
-- PostgreSQL 16+

-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ==========================================================================
-- users
-- ==========================================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash TEXT NOT NULL,
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    alias VARCHAR(50) UNIQUE,
    avatar_url TEXT NOT NULL DEFAULT '',
    phone VARCHAR(32),
    phone_verified BOOLEAN NOT NULL DEFAULT FALSE,
    security_level INTEGER NOT NULL DEFAULT 0,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    deleted_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_alias ON users(alias) WHERE alias IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_security_level ON users(security_level);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- ==========================================================================
-- social_bindings
-- ==========================================================================
CREATE TABLE social_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(30) NOT NULL,
    provider_uid VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_name VARCHAR(255),
    provider_avatar TEXT,
    access_token TEXT,
    refresh_token TEXT,
    token_expiry TIMESTAMPTZ,
    raw_profile JSONB,
    bound_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_uid),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_social_bindings_user_id ON social_bindings(user_id);
CREATE INDEX idx_social_bindings_provider ON social_bindings(provider);
CREATE INDEX idx_social_bindings_provider_email ON social_bindings(provider_email) WHERE provider_email IS NOT NULL;

-- ==========================================================================
-- phone_verifications
-- ==========================================================================
CREATE TABLE phone_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    phone VARCHAR(32) NOT NULL,
    code VARCHAR(10) NOT NULL,
    purpose VARCHAR(30) NOT NULL DEFAULT 'verify',
    attempts INTEGER NOT NULL DEFAULT 0,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_phone_verifications_phone ON phone_verifications(phone);
CREATE INDEX idx_phone_verifications_user_id ON phone_verifications(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_phone_verifications_expires_at ON phone_verifications(expires_at);

-- ==========================================================================
-- security_level_rules
-- ==========================================================================
CREATE TABLE security_level_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    level INTEGER NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_security_level_rules_level ON security_level_rules(level);
CREATE INDEX idx_security_level_rules_active ON security_level_rules(is_active, level DESC, priority DESC);

-- ==========================================================================
-- security_level_changes (audit)
-- ==========================================================================
CREATE TABLE security_level_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_level INTEGER NOT NULL,
    new_level INTEGER NOT NULL,
    reason VARCHAR(255) NOT NULL DEFAULT '',
    rule_id UUID REFERENCES security_level_rules(id) ON DELETE SET NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_security_level_changes_user_id ON security_level_changes(user_id, created_at DESC);
CREATE INDEX idx_security_level_changes_created_at ON security_level_changes(created_at DESC);

-- ==========================================================================
-- oidc_clients
-- ==========================================================================
CREATE TABLE oidc_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(255) NOT NULL UNIQUE,
    client_secret_hash TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    homepage_url TEXT NOT NULL DEFAULT '',
    redirect_uris TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    post_logout_redirect_uris TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    grant_types TEXT[] NOT NULL DEFAULT ARRAY['authorization_code', 'refresh_token']::TEXT[],
    response_types TEXT[] NOT NULL DEFAULT ARRAY['code']::TEXT[],
    scopes TEXT[] NOT NULL DEFAULT ARRAY['openid', 'profile', 'email']::TEXT[],
    audience TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    token_endpoint_auth_method VARCHAR(50) NOT NULL DEFAULT 'client_secret_basic',
    protocol_type VARCHAR(20) NOT NULL DEFAULT 'oidc',
    min_security_level INTEGER NOT NULL DEFAULT 0,
    require_pkce BOOLEAN NOT NULL DEFAULT TRUE,
    require_consent BOOLEAN NOT NULL DEFAULT TRUE,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    is_first_party BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oidc_clients_client_id ON oidc_clients(client_id);
CREATE INDEX idx_oidc_clients_owner_id ON oidc_clients(owner_id) WHERE owner_id IS NOT NULL;
CREATE INDEX idx_oidc_clients_active ON oidc_clients(is_active);

-- ==========================================================================
-- client_access_rules
-- ==========================================================================
CREATE TABLE client_access_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    rule_type VARCHAR(30) NOT NULL,
    rule_value TEXT NOT NULL,
    effect VARCHAR(10) NOT NULL DEFAULT 'allow',
    priority INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_client_access_rules_client_id ON client_access_rules(client_id);
CREATE INDEX idx_client_access_rules_active ON client_access_rules(client_id, is_active, priority DESC);

-- ==========================================================================
-- OAuth2 / OIDC storage tables (for fosite)
-- ==========================================================================
CREATE TABLE oauth2_authorization_codes (
    signature VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_id VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    granted_scopes TEXT NOT NULL DEFAULT '',
    form_data TEXT NOT NULL DEFAULT '',
    session_data BYTEA,
    subject VARCHAR(255) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    requested_audience TEXT NOT NULL DEFAULT '',
    granted_audience TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_auth_codes_request_id ON oauth2_authorization_codes(request_id);
CREATE INDEX idx_oauth2_auth_codes_client_id ON oauth2_authorization_codes(client_id);
CREATE INDEX idx_oauth2_auth_codes_subject ON oauth2_authorization_codes(subject);
CREATE INDEX idx_oauth2_auth_codes_expires_at ON oauth2_authorization_codes(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE oauth2_access_tokens (
    signature VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_id VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    granted_scopes TEXT NOT NULL DEFAULT '',
    form_data TEXT NOT NULL DEFAULT '',
    session_data BYTEA,
    subject VARCHAR(255) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    requested_audience TEXT NOT NULL DEFAULT '',
    granted_audience TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_access_tokens_request_id ON oauth2_access_tokens(request_id);
CREATE INDEX idx_oauth2_access_tokens_client_id ON oauth2_access_tokens(client_id);
CREATE INDEX idx_oauth2_access_tokens_subject ON oauth2_access_tokens(subject);
CREATE INDEX idx_oauth2_access_tokens_expires_at ON oauth2_access_tokens(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE oauth2_refresh_tokens (
    signature VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_id VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    granted_scopes TEXT NOT NULL DEFAULT '',
    form_data TEXT NOT NULL DEFAULT '',
    session_data BYTEA,
    subject VARCHAR(255) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    requested_audience TEXT NOT NULL DEFAULT '',
    granted_audience TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_refresh_tokens_request_id ON oauth2_refresh_tokens(request_id);
CREATE INDEX idx_oauth2_refresh_tokens_client_id ON oauth2_refresh_tokens(client_id);
CREATE INDEX idx_oauth2_refresh_tokens_subject ON oauth2_refresh_tokens(subject);

CREATE TABLE oauth2_oidc_sessions (
    signature VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_id VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    granted_scopes TEXT NOT NULL DEFAULT '',
    form_data TEXT NOT NULL DEFAULT '',
    session_data BYTEA,
    subject VARCHAR(255) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    requested_audience TEXT NOT NULL DEFAULT '',
    granted_audience TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_oidc_sessions_request_id ON oauth2_oidc_sessions(request_id);
CREATE INDEX idx_oauth2_oidc_sessions_subject ON oauth2_oidc_sessions(subject);

CREATE TABLE oauth2_pkce_requests (
    signature VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_id VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    granted_scopes TEXT NOT NULL DEFAULT '',
    form_data TEXT NOT NULL DEFAULT '',
    session_data BYTEA,
    subject VARCHAR(255) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    requested_audience TEXT NOT NULL DEFAULT '',
    granted_audience TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_pkce_request_id ON oauth2_pkce_requests(request_id);

-- ==========================================================================
-- signing_keys
-- ==========================================================================
CREATE TABLE signing_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kid VARCHAR(64) NOT NULL UNIQUE,
    algorithm VARCHAR(20) NOT NULL DEFAULT 'RS256',
    use_type VARCHAR(20) NOT NULL DEFAULT 'sig',
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    activated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    rotated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signing_keys_kid ON signing_keys(kid);
CREATE INDEX idx_signing_keys_current ON signing_keys(is_current) WHERE is_current = TRUE;
CREATE UNIQUE INDEX idx_signing_keys_only_one_current ON signing_keys(is_current) WHERE is_current = TRUE;

-- ==========================================================================
-- user_sessions
-- ==========================================================================
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    device_info JSONB,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token ON user_sessions(token);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- ==========================================================================
-- provider_configs
-- ==========================================================================
CREATE TABLE provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(30) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    client_id TEXT NOT NULL DEFAULT '',
    client_secret TEXT NOT NULL DEFAULT '',
    scopes TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    redirect_path VARCHAR(255) NOT NULL DEFAULT '',
    extra_config JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_provider_configs_provider ON provider_configs(provider);
CREATE INDEX idx_provider_configs_enabled ON provider_configs(enabled);

-- ==========================================================================
-- global_settings
-- ==========================================================================
CREATE TABLE global_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================================================
-- alias_restrictions
-- ==========================================================================
CREATE TABLE alias_restrictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern VARCHAR(255) NOT NULL UNIQUE,
    restriction_type VARCHAR(20) NOT NULL DEFAULT 'reserved',
    description TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alias_restrictions_pattern ON alias_restrictions(pattern);
CREATE INDEX idx_alias_restrictions_type ON alias_restrictions(restriction_type);

-- ==========================================================================
-- audit_log
-- ==========================================================================
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL DEFAULT '',
    resource_id VARCHAR(255) NOT NULL DEFAULT '',
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    error_message TEXT NOT NULL DEFAULT '',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_user_id ON audit_log(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_log_actor_id ON audit_log(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at DESC);
CREATE INDEX idx_audit_log_resource ON audit_log(resource_type, resource_id);
