package port

import "context"

// EmailSender sends emails (verification codes, password resets, etc.).
type EmailSender interface {
	SendRegistrationCode(ctx context.Context, to, code string) error
	SendVerificationEmail(ctx context.Context, to, token string) error
	SendPasswordResetEmail(ctx context.Context, to, token string) error
	SendRiskReportResolved(ctx context.Context, to, reportID, outcome, reason string) error
}
