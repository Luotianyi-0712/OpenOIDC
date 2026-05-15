package port

import "context"

// EmailSender sends emails (verification codes, password resets, etc.).
type EmailSender interface {
	SendVerificationEmail(ctx context.Context, to, token string) error
	SendPasswordResetEmail(ctx context.Context, to, token string) error
}
