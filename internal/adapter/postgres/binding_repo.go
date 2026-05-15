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

const bindingColumns = `id, user_id, provider, provider_uid, provider_email, provider_name,
	access_token, refresh_token, token_expiry, raw_profile, bound_at, verified_at, created_at, updated_at`

func scanBinding(row pgx.Row) (*domain.SocialBinding, error) {
	var b domain.SocialBinding
	var providerEmail, providerName, accessToken, refreshToken sql.NullString
	var tokenExpiry, verifiedAt sql.NullTime
	var rawProfile []byte

	err := row.Scan(
		&b.ID, &b.UserID, &b.Provider, &b.ProviderUID, &providerEmail, &providerName,
		&accessToken, &refreshToken, &tokenExpiry, &rawProfile,
		&b.BoundAt, &verifiedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if providerEmail.Valid {
		b.ProviderEmail = &providerEmail.String
	}
	if providerName.Valid {
		b.ProviderName = &providerName.String
	}
	if accessToken.Valid {
		b.AccessToken = &accessToken.String
	}
	if refreshToken.Valid {
		b.RefreshToken = &refreshToken.String
	}
	if tokenExpiry.Valid {
		b.TokenExpiry = &tokenExpiry.Time
	}
	if verifiedAt.Valid {
		b.VerifiedAt = &verifiedAt.Time
	}
	if len(rawProfile) > 0 {
		_ = json.Unmarshal(rawProfile, &b.RawProfile)
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

	var rawProfile []byte
	if b.RawProfile != nil {
		var err error
		rawProfile, err = json.Marshal(b.RawProfile)
		if err != nil {
			return fmt.Errorf("marshal raw_profile: %w", err)
		}
	}

	query := `
		INSERT INTO social_bindings (
			id, user_id, provider, provider_uid, provider_email, provider_name,
			access_token, refresh_token, token_expiry, raw_profile,
			bound_at, verified_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`
	_, err := r.db.Exec(ctx, query,
		b.ID, b.UserID, b.Provider, b.ProviderUID,
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
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE provider = $1 AND provider_uid = $2`
	row := r.db.QueryRow(ctx, query, provider, uid)
	b, err := scanBinding(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BindingRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error) {
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = $1 ORDER BY bound_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
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
	query := `SELECT ` + bindingColumns + ` FROM social_bindings WHERE user_id = $1 AND provider = $2`
	row := r.db.QueryRow(ctx, query, userID, provider)
	b, err := scanBinding(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BindingRepo) Delete(ctx context.Context, userID uuid.UUID, provider string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM social_bindings WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *BindingRepo) UpdateTokens(ctx context.Context, id uuid.UUID, access, refresh string, expiry *time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE social_bindings SET access_token = $2, refresh_token = $3, token_expiry = $4, updated_at = NOW() WHERE id = $1`,
		id, access, refresh, toNullTime(expiry),
	)
	return err
}
