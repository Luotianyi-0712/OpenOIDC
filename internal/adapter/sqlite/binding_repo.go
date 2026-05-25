package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type BindingRepo struct {
	db *sql.DB
}

func NewBindingRepo(db *sql.DB) *BindingRepo {
	return &BindingRepo{db: db}
}

const bindingColumns = `id, user_id, provider, provider_uid, provider_email, provider_name, provider_avatar,
	status, access_token, refresh_token, token_expiry, token_type, token_scopes, raw_profile,
	bound_at, verified_at, unbound_at, unbind_reason, last_auth_check_at, last_auth_status, last_auth_error,
	created_at, updated_at`

func scanBinding(row interface{ Scan(dest ...any) error }) (*domain.SocialBinding, error) {
	var b domain.SocialBinding
	var id, userID string
	var providerEmail, providerName, providerAvatar, accessToken, refreshToken, tokenType sql.NullString
	var tokenScopes, rawProfile, status, lastAuthStatus sql.NullString
	var unbindReason, lastAuthError sql.NullString
	var tokenExpiry, verifiedAt, unboundAt, lastAuthCheckAt sql.NullTime

	err := row.Scan(
		&id, &userID, &b.Provider, &b.ProviderUID, &providerEmail, &providerName, &providerAvatar,
		&status, &accessToken, &refreshToken, &tokenExpiry, &tokenType, &tokenScopes, &rawProfile,
		&b.BoundAt, &verifiedAt, &unboundAt, &unbindReason, &lastAuthCheckAt, &lastAuthStatus, &lastAuthError,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	b.ID = uuid.MustParse(id)
	b.UserID = uuid.MustParse(userID)
	b.ProviderEmail = fromNullString(providerEmail)
	b.ProviderName = fromNullString(providerName)
	b.ProviderAvatar = fromNullString(providerAvatar)
	b.Status = domain.SocialBindingStatus(status.String)
	if b.Status == "" {
		b.Status = domain.SocialBindingStatusActive
	}
	b.AccessToken = fromNullString(accessToken)
	b.RefreshToken = fromNullString(refreshToken)
	b.TokenExpiry = fromNullTime(tokenExpiry)
	b.TokenType = fromNullString(tokenType)
	b.VerifiedAt = fromNullTime(verifiedAt)
	b.UnboundAt = fromNullTime(unboundAt)
	b.UnbindReason = fromNullString(unbindReason)
	b.LastAuthCheckAt = fromNullTime(lastAuthCheckAt)
	b.LastAuthStatus = domain.SocialAuthStatus(lastAuthStatus.String)
	if b.LastAuthStatus == "" {
		b.LastAuthStatus = domain.SocialAuthStatusUnknown
	}
	b.LastAuthError = fromNullString(lastAuthError)

	if tokenScopes.Valid && tokenScopes.String != "" {
		_ = json.Unmarshal([]byte(tokenScopes.String), &b.TokenScopes)
	}
	if rawProfile.Valid && rawProfile.String != "" {
		_ = json.Unmarshal([]byte(rawProfile.String), &b.RawProfile)
	}
	return &b, nil
}

func (r *BindingRepo) Create(ctx context.Context, b *domain.SocialBinding) error {
	prepareBindingForWrite(b)
	rawProfile, err := marshalJSON(b.RawProfile)
	if err != nil {
		return fmt.Errorf("marshal raw_profile: %w", err)
	}
	tokenScopes, err := marshalJSON(b.TokenScopes)
	if err != nil {
		return fmt.Errorf("marshal token_scopes: %w", err)
	}

	query := `
		INSERT INTO social_bindings (
			id, user_id, provider, provider_uid, provider_email, provider_name, provider_avatar,
			status, access_token, refresh_token, token_expiry, token_type, token_scopes, raw_profile,
			bound_at, verified_at, unbound_at, unbind_reason, last_auth_check_at, last_auth_status, last_auth_error,
			created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`
	_, err = r.db.ExecContext(ctx, query,
		b.ID.String(), b.UserID.String(), b.Provider, b.ProviderUID,
		toNullString(b.ProviderEmail), toNullString(b.ProviderName), toNullString(b.ProviderAvatar),
		string(b.Status), toNullString(b.AccessToken), toNullString(b.RefreshToken),
		toNullTime(b.TokenExpiry), toNullString(b.TokenType), toNullJSON(tokenScopes), toNullJSON(rawProfile),
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
	b.UpdatedAt = time.Now().UTC()
	rawProfile, err := marshalJSON(b.RawProfile)
	if err != nil {
		return fmt.Errorf("marshal raw_profile: %w", err)
	}
	tokenScopes, err := marshalJSON(b.TokenScopes)
	if err != nil {
		return fmt.Errorf("marshal token_scopes: %w", err)
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE social_bindings SET
			user_id = ?, provider = ?, provider_uid = ?, provider_email = ?, provider_name = ?, provider_avatar = ?,
			status = ?, access_token = ?, refresh_token = ?, token_expiry = ?, token_type = ?, token_scopes = ?, raw_profile = ?,
			bound_at = ?, verified_at = ?, unbound_at = ?, unbind_reason = ?,
			last_auth_check_at = ?, last_auth_status = ?, last_auth_error = ?, updated_at = ?
		WHERE id = ?`,
		b.UserID.String(), b.Provider, b.ProviderUID,
		toNullString(b.ProviderEmail), toNullString(b.ProviderName), toNullString(b.ProviderAvatar),
		string(b.Status), toNullString(b.AccessToken), toNullString(b.RefreshToken),
		toNullTime(b.TokenExpiry), toNullString(b.TokenType), toNullJSON(tokenScopes), toNullJSON(rawProfile),
		b.BoundAt, toNullTime(b.VerifiedAt), toNullTime(b.UnboundAt), toNullString(b.UnbindReason),
		toNullTime(b.LastAuthCheckAt), string(b.LastAuthStatus), toNullString(b.LastAuthError), b.UpdatedAt,
		b.ID.String(),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *BindingRepo) GetByProviderUID(ctx context.Context, provider, uid string) (*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE provider = ? AND provider_uid = ? AND status = ? ORDER BY bound_at DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, provider, uid, domain.SocialBindingStatusActive)
	return scanBindingNotFound(row)
}

func (r *BindingRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = ? AND status = ? ORDER BY bound_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), domain.SocialBindingStatusActive)
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
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = ? AND provider = ? AND status = ? ORDER BY bound_at DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, userID.String(), provider, domain.SocialBindingStatusActive)
	return scanBindingNotFound(row)
}

func (r *BindingRepo) SoftUnbind(ctx context.Context, userID uuid.UUID, provider, reason string) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE social_bindings
		SET status = ?, unbound_at = ?, unbind_reason = ?,
			access_token = NULL, refresh_token = NULL, token_expiry = NULL, token_type = NULL, token_scopes = NULL,
			updated_at = ?
		WHERE user_id = ? AND provider = ? AND status = ?`,
		domain.SocialBindingStatusUserUnbound, now, reason, now,
		userID.String(), provider, domain.SocialBindingStatusActive,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
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
		WHERE status = ?
		  AND last_auth_status <> ?
		  AND (last_auth_check_at IS NULL OR last_auth_check_at <= ?)
		ORDER BY COALESCE(last_auth_check_at, '1970-01-01T00:00:00Z') ASC
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, domain.SocialBindingStatusActive, domain.SocialAuthStatusUnsupported, before, limit)
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

func scanBindingNotFound(row interface{ Scan(dest ...any) error }) (*domain.SocialBinding, error) {
	b, err := scanBinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	b.UpdatedAt = now
}

func marshalJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func toNullJSON(data []byte) sql.NullString {
	if len(data) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(data), Valid: true}
}

func isDuplicateSocialBindingConstraint(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}
