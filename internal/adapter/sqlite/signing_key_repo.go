package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

// SigningKeyRepo implements port.SigningKeyRepository using SQLite.
type SigningKeyRepo struct {
	db *sql.DB
}

// NewSigningKeyRepo returns a new SigningKeyRepo.
func NewSigningKeyRepo(db *sql.DB) *SigningKeyRepo {
	return &SigningKeyRepo{db: db}
}

// Create inserts a new signing key.
func (r *SigningKeyRepo) Create(ctx context.Context, k *domain.SigningKey) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO signing_keys
		 (id, key_id, algorithm, private_key, public_key, is_current, created_at, rotated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID.String(), k.KeyID, k.Algorithm, k.PrivateKey, k.PublicKey,
		k.IsCurrent, k.CreatedAt, toNullTime(k.RotatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert signing key: %w", err)
	}
	return nil
}

func scanSigningKey(row interface{ Scan(...any) error }) (*domain.SigningKey, error) {
	var k domain.SigningKey
	var id string
	var rotatedAt sql.NullTime

	if err := row.Scan(
		&id, &k.KeyID, &k.Algorithm, &k.PrivateKey, &k.PublicKey,
		&k.IsCurrent, &k.CreatedAt, &rotatedAt,
	); err != nil {
		return nil, err
	}
	k.ID = uuid.MustParse(id)
	if rotatedAt.Valid {
		k.RotatedAt = &rotatedAt.Time
	}
	return &k, nil
}

// GetCurrent returns the currently active signing key.
func (r *SigningKeyRepo) GetCurrent(ctx context.Context) (*domain.SigningKey, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, key_id, algorithm, private_key, public_key, is_current, created_at, rotated_at
		 FROM signing_keys WHERE is_current = 1 LIMIT 1`,
	)
	k, err := scanSigningKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return k, nil
}

// List returns all signing keys ordered by created_at descending.
func (r *SigningKeyRepo) List(ctx context.Context) ([]*domain.SigningKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, key_id, algorithm, private_key, public_key, is_current, created_at, rotated_at
		 FROM signing_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.SigningKey
	for rows.Next() {
		k, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Rotate deactivates the old key and activates the new key in a single transaction.
func (r *SigningKeyRepo) Rotate(ctx context.Context, oldID, newID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	if _, err := tx.ExecContext(ctx,
		`UPDATE signing_keys SET is_current = 0, rotated_at = ? WHERE id = ?`,
		now, oldID.String(),
	); err != nil {
		return fmt.Errorf("deactivate old key: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE signing_keys SET is_current = 1 WHERE id = ?`,
		newID.String(),
	); err != nil {
		return fmt.Errorf("activate new key: %w", err)
	}

	return tx.Commit()
}
