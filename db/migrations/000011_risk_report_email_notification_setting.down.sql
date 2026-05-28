-- Remove risk report email notification global setting
DELETE FROM global_settings WHERE key = 'risk_report_email_notification_enabled';
