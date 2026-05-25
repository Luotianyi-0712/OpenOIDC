DROP INDEX IF EXISTS idx_social_bindings_active_user_provider;
DROP INDEX IF EXISTS idx_social_bindings_active_provider_uid;
DROP INDEX IF EXISTS idx_social_bindings_auth_due;

ALTER TABLE social_bindings DROP COLUMN IF EXISTS last_auth_error;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS last_auth_status;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS last_auth_check_at;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS unbind_reason;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS unbound_at;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS token_scopes;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS token_type;
ALTER TABLE social_bindings DROP COLUMN IF EXISTS status;

ALTER TABLE social_bindings ADD CONSTRAINT social_bindings_provider_provider_uid_key UNIQUE (provider, provider_uid);
ALTER TABLE social_bindings ADD CONSTRAINT social_bindings_user_id_provider_key UNIQUE (user_id, provider);