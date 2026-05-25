-- Restore previous default for require_email_verified.
ALTER TABLE oidc_clients ALTER COLUMN require_email_verified SET DEFAULT FALSE;