package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BindingRepo struct {
	db *pgxpool.Pool
}

func NewBindingRepo(db *pgxpool.Pool) *BindingRepo {
	return &BindingRepo{db: db}
}

const bindingColumns = `id, user_id, provider, provider_uid, provider_email, provider_name, provider_avatar,
	status, access_token, refresh_token, token_expiry, token_type, token_scopes, raw_profile,
	bound_at, verified_at, unbound_at, unbind_reason, last_auth_check_at, last_auth_status, last_auth_error,
	created_at, updated_at`

func scanBinding(row pgx.Row) (*domain.SocialBinding, error) {
	var b domain.SocialBinding
	var providerEmail, providerName, providerAvatar, accessToken, refreshToken, tokenType sql.NullString
	var unbindReason, lastAuthError sql.NullString
	var tokenExpiry, verifiedAt, unboundAt, lastAuthCheckAt sql.NullTime
	var status, lastAuthStatus string
	var rawProfile []byte
	var tokenScopes []string

	err := row.Scan(
		&b.ID, &b.UserID, &b.Provider, &b.ProviderUID, &providerEmail, &providerName, &providerAvatar,
		&status, &accessToken, &refreshToken, &tokenExpiry, &tokenType, &tokenScopes, &rawProfile,
		&b.BoundAt, &verifiedAt, &unboundAt, &unbindReason, &lastAuthCheckAt, &lastAuthStatus, &lastAuthError,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	b.ProviderEmail = fromNullString(providerEmail)
	b.ProviderName = fromNullString(providerName)
	b.ProviderAvatar = fromNullString(providerAvatar)
	b.Status = domain.SocialBindingStatus(status)
	if b.Status == "" {
		b.Status = domain.SocialBindingStatusActive
	}
	b.AccessToken = fromNullString(accessToken)
	b.RefreshToken = fromNullString(refreshToken)
	b.TokenExpiry = fromNullTime(tokenExpiry)
	b.TokenType = fromNullString(tokenType)
	b.TokenScopes = tokenScopes
	b.VerifiedAt = fromNullTime(verifiedAt)
	b.UnboundAt = fromNullTime(unboundAt)
	b.UnbindReason = fromNullString(unbindReason)
	b.LastAuthCheckAt = fromNullTime(lastAuthCheckAt)
	b.LastAuthStatus = domain.SocialAuthStatus(lastAuthStatus)
	if b.LastAuthStatus == "" {
		b.LastAuthStatus = domain.SocialAuthStatusUnknown
	}
	b.LastAuthError = fromNullString(lastAuthError)
	if len(rawProfile) > 0 {
		_ = json.Unmarshal(rawProfile, &b.RawProfile)
	}
	return &b, nil
}

func (r *BindingRepo) Create(ctx context.Context, b *domain.SocialBinding) error {
	prepareBindingForWrite(b)
	rawProfile, err := marshalRawProfile(b.RawProfile)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO social_bindings (
			id, user_id, provider, provider_uid, provider_email, provider_name, provider_avatar,
			status, access_token, refresh_token, token_expiry, token_type, token_scopes, raw_profile,
			bound_at, verified_at, unbound_at, unbind_reason, last_auth_check_at, last_auth_status, last_auth_error,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
	`
	_, err = r.db.Exec(ctx, query,
		b.ID, b.UserID, b.Provider, b.ProviderUID,
		toNullString(b.ProviderEmail), toNullString(b.ProviderName), toNullString(b.ProviderAvatar),
		string(b.Status), toNullString(b.AccessToken), toNullString(b.RefreshToken),
		toNullTime(b.TokenExpiry), toNullString(b.TokenType), b.TokenScopes, rawProfile,
		b.BoundAt, toNullTime(b.VerifiedAt), toNullTime(b.UnboundAt), toNullString(b.UnbindReason),
		toNullTime(b.LastAuthCheckAt), string(b.LastAuthStatus), toNullString(b.LastAuthError),
		b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert binding: %w", err)
	}
	return nil
}

func (r *BindingRepo) Update(ctx context.Context, b *domain.SocialBinding) error {
	if b.ID == uuid.Nil {
		return fmt.Errorf("binding id required")
	}
	if b.Status == "" {
		b.Status = domain.SocialBindingStatusActive
	}
	if b.LastAuthStatus == "" {
		b.LastAuthStatus = domain.SocialAuthStatusUnknown
	}
	if b.TokenScopes == nil {
		b.TokenScopes = []string{}
	}
	b.UpdatedAt = time.Now().UTC()
	rawProfile, err := marshalRawProfile(b.RawProfile)
	if err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE social_bindings SET
			user_id = $2, provider = $3, provider_uid = $4, provider_email = $5, provider_name = $6, provider_avatar = $7,
			status = $8, access_token = $9, refresh_token = $10, token_expiry = $11, token_type = $12, token_scopes = $13, raw_profile = $14,
			bound_at = $15, verified_at = $16, unbound_at = $17, unbind_reason = $18,
			last_auth_check_at = $19, last_auth_status = $20, last_auth_error = $21, updated_at = $22
		WHERE id = $1`,
		b.ID, b.UserID, b.Provider, b.ProviderUID,
		toNullString(b.ProviderEmail), toNullString(b.ProviderName), toNullString(b.ProviderAvatar),
		string(b.Status), toNullString(b.AccessToken), toNullString(b.RefreshToken),
		toNullTime(b.TokenExpiry), toNullString(b.TokenType), b.TokenScopes, rawProfile,
		b.BoundAt, toNullTime(b.VerifiedAt), toNullTime(b.UnboundAt), toNullString(b.UnbindReason),
		toNullTime(b.LastAuthCheckAt), string(b.LastAuthStatus), toNullString(b.LastAuthError), b.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *BindingRepo) GetByProviderUID(ctx context.Context, provider, uid string) (*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE provider = $1 AND provider_uid = $2 AND status = $3 ORDER BY bound_at DESC LIMIT 1`
	row := r.db.QueryRow(ctx, query, provider, uid, domain.SocialBindingStatusActive)
	return scanBindingNotFound(row)
}

func (r *BindingRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = $1 AND status = $2 ORDER BY bound_at DESC`
	rows, err := r.db.Query(ctx, query, userID, domain.SocialBindingStatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []*domain.SocialBinding
	for rows.Next() {
		b, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, b)
	}
	return bindings, rows.Err()
}

func (r *BindingRepo) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = $1 AND provider = $2 AND status = $3 ORDER BY bound_at DESC LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID, provider, domain.SocialBindingStatusActive)
	return scanBindingNotFound(row)
}

func (r *BindingRepo) SoftUnbind(ctx context.Context, userID uuid.UUID, provider, reason string) error {
	now := time.Now().UTC()
	status := domain.SocialBindingStatusUserUnbound
	tag, err := r.db.Exec(ctx, `
		UPDATE social_bindings
		SET status = $3, unbound_at = $4, unbind_reason = $5,
			access_token = NULL, refresh_token = NULL, token_expiry = NULL, token_type = NULL, token_scopes = ARRAY[]::TEXT[],
			updated_at = $4
		WHERE user_id = $1 AND provider = $2 AND status = $6`,
		userID, provider, status, now, reason, domain.SocialBindingStatusActive,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *BindingRepo) ListDueAuthChecks(ctx context.Context, before time.Time, limit int) ([]*domain.SocialBinding, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT ` + bindingColumns + `
		FROM social_bindings
		WHERE status = $1
		  AND last_auth_status <> $2
		  AND (last_auth_check_at IS NULL OR last_auth_check_at <= $3)
		ORDER BY COALESCE(last_auth_check_at, TIMESTAMPTZ 'epoch') ASC
		LIMIT $4`
	rows, err := r.db.Query(ctx, query, domain.SocialBindingStatusActive, domain.SocialAuthStatusUnsupported, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bindings []*domain.SocialBinding
	for rows.Next() {
		b, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, b)
	}
	return bindings, rows.Err()
}

func scanBindingNotFound(row pgx.Row) (*domain.SocialBinding, error) {
	b, err := scanBinding(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func prepareBindingForWrite(b *domain.SocialBinding) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.BoundAt.IsZero() {
		b.BoundAt = now
	}
	if b.Status == "" {
		b.Status = domain.SocialBindingStatusActive
	}
	if b.LastAuthStatus == "" {
		b.LastAuthStatus = domain.SocialAuthStatusUnknown
	}
	if b.TokenScopes == nil {
		b.TokenScopes = []string{}
	}
	b.UpdatedAt = now
}

func marshalRawProfile(raw map[string]any) ([]byte, error) {
	if raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw_profile: %w", err)
	}
	return data, nil
}
