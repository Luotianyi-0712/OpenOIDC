-- Reverse migration: remove risk tables and added columns

DROP TABLE IF EXISTS risk_list;
DROP TABLE IF EXISTS risk_reports;

ALTER TABLE oidc_clients DROP COLUMN IF EXISTS client_secret_plain;
ALTER TABLE oidc_clients DROP COLUMN IF EXISTS require_email_verified;
ALTER TABLE oidc_clients DROP COLUMN IF EXISTS is_confidential;
