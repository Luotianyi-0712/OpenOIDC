-- Down migration: drop all tables in dependency order

DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS alias_restrictions CASCADE;
DROP TABLE IF EXISTS global_settings CASCADE;
DROP TABLE IF EXISTS provider_configs CASCADE;
DROP TABLE IF EXISTS user_sessions CASCADE;
DROP TABLE IF EXISTS signing_keys CASCADE;
DROP TABLE IF EXISTS oauth2_pkce_requests CASCADE;
DROP TABLE IF EXISTS oauth2_oidc_sessions CASCADE;
DROP TABLE IF EXISTS oauth2_refresh_tokens CASCADE;
DROP TABLE IF EXISTS oauth2_access_tokens CASCADE;
DROP TABLE IF EXISTS oauth2_authorization_codes CASCADE;
DROP TABLE IF EXISTS client_access_rules CASCADE;
DROP TABLE IF EXISTS oidc_clients CASCADE;
DROP TABLE IF EXISTS security_level_changes CASCADE;
DROP TABLE IF EXISTS security_level_rules CASCADE;
DROP TABLE IF EXISTS phone_verifications CASCADE;
DROP TABLE IF EXISTS social_bindings CASCADE;
DROP TABLE IF EXISTS users CASCADE;
