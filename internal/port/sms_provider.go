package port

import "context"

type SMSProvider interface {
	SendCode(ctx context.Context, phoneNumber string, code string) error
}
