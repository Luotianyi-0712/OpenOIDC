package sqlite

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
)

type BindingRepo struct {
	db *sql.DB
}

func NewBindingRepo(db *sql.DB) *BindingRepo {
	return &BindingRepo{db: db}
}

const bindingColumns = `id, user_id, provider, provider_uid, provider_email, provider_name,
	access_token, refresh_token, token_expiry, raw_profile, bound_at, verified_at, created_at, updated_at`

func scanBinding(row interface{ Scan(dest ...any) error }) (*domain.SocialBinding, error) {
	var b domain.SocialBinding
	var id, userID string
	var providerEmail, providerName, accessToken, refreshToken sql.NullString
	var tokenExpiry, verifiedAt sql.NullTime
	var rawProfile sql.NullString

	err := row.Scan(
		&id, &userID, &b.Provider, &b.ProviderUID, &providerEmail, &providerName,
		&accessToken, &refreshToken, &tokenExpiry, &rawProfile,
		&b.BoundAt, &verifiedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	b.ID = uuid.MustParse(id)
	b.UserID = uuid.MustParse(userID)
	b.ProviderEmail = fromNullString(providerEmail)
	b.ProviderName = fromNullString(providerName)
	b.AccessToken = fromNullString(accessToken)
	b.RefreshToken = fromNullString(refreshToken)
	b.TokenExpiry = fromNullTime(tokenExpiry)
	b.VerifiedAt = fromNullTime(verifiedAt)

	if rawProfile.Valid && rawProfile.String != "" {
		_ = json.Unmarshal([]byte(rawProfile.String), &b.RawProfile)
	}
	return &b, nil
}

func (r *BindingRepo) Create(ctx context.Context, b *domain.SocialBinding) error {
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
	b.UpdatedAt = now

	var rawProfile sql.NullString
	if b.RawProfile != nil {
		data, err := json.Marshal(b.RawProfile)
		if err != nil {
			return fmt.Errorf("marshal raw_profile: %w", err)
		}
		rawProfile = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO social_bindings (
			id, user_id, provider, provider_uid, provider_email, provider_name,
			access_token, refresh_token, token_expiry, raw_profile,
			bound_at, verified_at, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`
	_, err := r.db.ExecContext(ctx, query,
		b.ID.String(), b.UserID.String(), b.Provider, b.ProviderUID,
		toNullString(b.ProviderEmail), toNullString(b.ProviderName),
		toNullString(b.AccessToken), toNullString(b.RefreshToken),
		toNullTime(b.TokenExpiry), rawProfile,
		b.BoundAt, toNullTime(b.VerifiedAt), b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert binding: %w", err)
	}
	return nil
}

func (r *BindingRepo) GetByProviderUID(ctx context.Context, provider, uid string) (*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE provider = ? AND provider_uid = ?`
	row := r.db.QueryRowContext(ctx, query, provider, uid)
	b, err := scanBinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BindingRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = ? ORDER BY bound_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID.String())
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
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = ? AND provider = ?`
	row := r.db.QueryRowContext(ctx, query, userID.String(), provider)
	b, err := scanBinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BindingRepo) Delete(ctx context.Context, userID uuid.UUID, provider string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM social_bindings WHERE user_id = ? AND provider = ?`,
		userID.String(), provider,
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

func (r *BindingRepo) UpdateTokens(ctx context.Context, id uuid.UUID, access, refresh string, expiry *time.Time) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE social_bindings SET access_token = ?, refresh_token = ?, token_expiry = ?, updated_at = ? WHERE id = ?`,
		access, refresh, toNullTime(expiry), now, id.String(),
	)
	return err
}
