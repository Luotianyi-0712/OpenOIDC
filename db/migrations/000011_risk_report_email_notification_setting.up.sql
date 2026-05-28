-- Add risk report email notification global setting
INSERT INTO global_settings (key, value, description, created_at, updated_at)
VALUES (
    'risk_report_email_notification_enabled',
    '"true"'::jsonb,
    '控制是否允许用户接收举报处理结果的邮件通知。关闭后，用户个人资料页将不显示邮件通知开关。',
    NOW(),
    NOW()
)
ON CONFLICT (key) DO NOTHING;
