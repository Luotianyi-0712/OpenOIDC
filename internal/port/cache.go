package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error

	SetOAuthState(ctx context.Context, state string, data []byte, ttl time.Duration) error
	GetOAuthState(ctx context.Context, state string) ([]byte, error)
	DeleteOAuthState(ctx context.Context, state string) error

	IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error)

	SetEmailVerifyToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	GetEmailVerifyToken(ctx context.Context, token string) (uuid.UUID, error)

	SetPasswordResetToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error)
}
