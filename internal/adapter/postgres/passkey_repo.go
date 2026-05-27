package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type PasskeyRepo struct {
	db *pgxpool.Pool
}

func NewPasskeyRepo(db *pgxpool.Pool) *PasskeyRepo {
	return &PasskeyRepo{db: db}
}

func (r *PasskeyRepo) Create(ctx context.Context, cred *domain.PasskeyCredential) error {
	if cred.ID == uuid.Nil {
		cred.ID = uuid.New()
	}
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = time.Now().UTC()
	}
	if cred.Transport == nil {
		cred.Transport = []string{}
	}
	if cred.AttestationType == "" {
		cred.AttestationType = "none"
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO passkey_credentials
		 (id, user_id, credential_id, public_key, attestation_type, transport, sign_count, aaguid, name, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		cred.ID, cred.UserID, cred.CredentialID, cred.PublicKey,
		cred.AttestationType, cred.Transport, cred.SignCount, cred.AAGUID,
		cred.Name, cred.CreatedAt,
	)
	return err
}

func (r *PasskeyRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.PasskeyCredential, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, credential_id, public_key, attestation_type, transport,
		        sign_count, aaguid, name, last_used_at, created_at
		 FROM passkey_credentials WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*domain.PasskeyCredential
	for rows.Next() {
		c, err := scanPasskeyCred(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (r *PasskeyRepo) GetByCredentialID(ctx context.Context, credentialID []byte) (*domain.PasskeyCredential, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, credential_id, public_key, attestation_type, transport,
		        sign_count, aaguid, name, last_used_at, created_at
		 FROM passkey_credentials WHERE credential_id = $1`, credentialID)
	c, err := scanPasskeyCred(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *PasskeyRepo) UpdateSignCount(ctx context.Context, id uuid.UUID, count uint32) error {
	_, err := r.db.Exec(ctx,
		`UPDATE passkey_credentials SET sign_count = $1 WHERE id = $2`, count, id)
	return err
}

func (r *PasskeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE passkey_credentials SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), id)
	return err
}

func (r *PasskeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM passkey_credentials WHERE id = $1`, id)
	return err
}

func (r *PasskeyRepo) Rename(ctx context.Context, id uuid.UUID, name string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE passkey_credentials SET name = $1 WHERE id = $2`, name, id)
	return err
}

func scanPasskeyCred(row pgx.Row) (*domain.PasskeyCredential, error) {
	var c domain.PasskeyCredential
	var lastUsed *time.Time
	err := row.Scan(
		&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey,
		&c.AttestationType, &c.Transport, &c.SignCount, &c.AAGUID,
		&c.Name, &lastUsed, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.LastUsedAt = lastUsed
	return &c, nil
}
