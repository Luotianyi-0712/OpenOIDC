ALTER TABLE social_bindings DROP CONSTRAINT IF EXISTS social_bindings_provider_provider_uid_key;
ALTER TABLE social_bindings DROP CONSTRAINT IF EXISTS social_bindings_user_id_provider_key;

ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS provider_avatar TEXT;
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'active';
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS token_type VARCHAR(40);
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS token_scopes TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[];
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS unbound_at TIMESTAMPTZ;
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS unbind_reason TEXT;
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS last_auth_check_at TIMESTAMPTZ;
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS last_auth_status VARCHAR(30) NOT NULL DEFAULT 'unknown';
ALTER TABLE social_bindings ADD COLUMN IF NOT EXISTS last_auth_error TEXT;

UPDATE social_bindings
SET status = 'active'
WHERE status IS NULL OR status = '';

UPDATE social_bindings
SET last_auth_status = 'unknown'
WHERE last_auth_status IS NULL OR last_auth_status = '';

CREATE INDEX IF NOT EXISTS idx_social_bindings_auth_due
    ON social_bindings(last_auth_check_at)
    WHERE status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS idx_social_bindings_active_provider_uid
    ON social_bindings(provider, provider_uid)
    WHERE status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS idx_social_bindings_active_user_provider
    ON social_bindings(user_id, provider)
    WHERE status = 'active';