-- Ensure newly inserted OIDC clients require verified email by default.
ALTER TABLE oidc_clients ALTER COLUMN require_email_verified SET DEFAULT TRUE;

-- Align existing rows created before is_confidential was stored separately.
UPDATE oidc_clients SET is_confidential = NOT is_public;