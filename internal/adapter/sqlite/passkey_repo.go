package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type PasskeyRepo struct {
	db *sql.DB
}

func NewPasskeyRepo(db *sql.DB) *PasskeyRepo {
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
	transportJSON, _ := json.Marshal(cred.Transport)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO passkey_credentials
		 (id, user_id, credential_id, public_key, attestation_type, transport, sign_count, aaguid, name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cred.ID.String(), cred.UserID.String(), cred.CredentialID, cred.PublicKey,
		cred.AttestationType, string(transportJSON), cred.SignCount, cred.AAGUID,
		cred.Name, cred.CreatedAt,
	)
	return err
}

func (r *PasskeyRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.PasskeyCredential, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, credential_id, public_key, attestation_type, transport,
		        sign_count, aaguid, name, last_used_at, created_at
		 FROM passkey_credentials WHERE user_id = ? ORDER BY created_at`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*domain.PasskeyCredential
	for rows.Next() {
		c, err := scanSqlitePasskeyCred(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (r *PasskeyRepo) GetByCredentialID(ctx context.Context, credentialID []byte) (*domain.PasskeyCredential, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, credential_id, public_key, attestation_type, transport,
		        sign_count, aaguid, name, last_used_at, created_at
		 FROM passkey_credentials WHERE credential_id = ?`, credentialID)
	c, err := scanSqlitePasskeyCred(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *PasskeyRepo) UpdateSignCount(ctx context.Context, id uuid.UUID, count uint32) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE passkey_credentials SET sign_count = ? WHERE id = ?`, count, id.String())
	return err
}

func (r *PasskeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE passkey_credentials SET last_used_at = ? WHERE id = ?`, time.Now().UTC(), id.String())
	return err
}

func (r *PasskeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM passkey_credentials WHERE id = ?`, id.String())
	return err
}

func (r *PasskeyRepo) Rename(ctx context.Context, id uuid.UUID, name string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE passkey_credentials SET name = ? WHERE id = ?`, name, id.String())
	return err
}

func scanSqlitePasskeyCred(row interface{ Scan(...any) error }) (*domain.PasskeyCredential, error) {
	var c domain.PasskeyCredential
	var id, userID string
	var transportStr string
	var lastUsed sql.NullTime

	err := row.Scan(
		&id, &userID, &c.CredentialID, &c.PublicKey,
		&c.AttestationType, &transportStr, &c.SignCount, &c.AAGUID,
		&c.Name, &lastUsed, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.ID = uuid.MustParse(id)
	c.UserID = uuid.MustParse(userID)
	c.LastUsedAt = fromNullTime(lastUsed)
	if transportStr != "" {
		_ = json.Unmarshal([]byte(transportStr), &c.Transport)
	}
	return &c, nil
}
