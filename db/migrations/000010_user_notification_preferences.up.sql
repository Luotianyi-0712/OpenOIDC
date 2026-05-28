-- Add notification preferences to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS risk_report_email_enabled BOOLEAN NOT NULL DEFAULT true;
