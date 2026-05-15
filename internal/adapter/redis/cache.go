package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	prefixOAuthState    = "oauth_state:"
	prefixRateLimit     = "rate_limit:"
	prefixEmailVerify   = "email_verify:"
	prefixPasswordReset = "pwd_reset:"
)

type Cache struct {
	client *redis.Client
}

func NewCache(ctx context.Context, cfg config.RedisConfig) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Cache{client: client}, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Client() *redis.Client {
	return c.client
}

// ---------------------------------------------------------------------------
// Generic
// ---------------------------------------------------------------------------

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// ---------------------------------------------------------------------------
// OAuth state
// ---------------------------------------------------------------------------

func (c *Cache) SetOAuthState(ctx context.Context, state string, data []byte, ttl time.Duration) error {
	return c.client.Set(ctx, prefixOAuthState+state, data, ttl).Err()
}

func (c *Cache) GetOAuthState(ctx context.Context, state string) ([]byte, error) {
	data, err := c.client.Get(ctx, prefixOAuthState+state).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) DeleteOAuthState(ctx context.Context, state string) error {
	return c.client.Del(ctx, prefixOAuthState+state).Err()
}

// ---------------------------------------------------------------------------
// Rate limit
// ---------------------------------------------------------------------------

func (c *Cache) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := prefixRateLimit + key
	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// ---------------------------------------------------------------------------
// Email verify tokens
// ---------------------------------------------------------------------------

func (c *Cache) SetEmailVerifyToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return c.client.Set(ctx, prefixEmailVerify+token, userID.String(), ttl).Err()
}

func (c *Cache) GetEmailVerifyToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := c.client.Get(ctx, prefixEmailVerify+token).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, port.ErrNotFound
		}
		return uuid.Nil, err
	}
	return uuid.Parse(val)
}

// ---------------------------------------------------------------------------
// Password reset tokens
// ---------------------------------------------------------------------------

func (c *Cache) SetPasswordResetToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return c.client.Set(ctx, prefixPasswordReset+token, userID.String(), ttl).Err()
}

func (c *Cache) GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := c.client.Get(ctx, prefixPasswordReset+token).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, port.ErrNotFound
		}
		return uuid.Nil, err
	}
	return uuid.Parse(val)
}
