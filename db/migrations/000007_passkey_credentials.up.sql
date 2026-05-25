CREATE TABLE passkey_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    attestation_type VARCHAR(30) NOT NULL DEFAULT 'none',
    transport TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    sign_count BIGINT NOT NULL DEFAULT 0,
    aaguid BYTEA,
    name VARCHAR(100) NOT NULL DEFAULT '',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_passkey_credentials_user_id ON passkey_credentials(user_id);