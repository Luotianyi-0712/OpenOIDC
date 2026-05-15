package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SigningKeyRepo struct {
	db *pgxpool.Pool
}

func NewSigningKeyRepo(db *pgxpool.Pool) *SigningKeyRepo {
	return &SigningKeyRepo{db: db}
}

func (r *SigningKeyRepo) Create(ctx context.Context, k *domain.SigningKey) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO signing_keys
		 (id, kid, algorithm, public_key, private_key, is_current, activated_at, rotated_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		k.ID, k.KeyID, k.Algorithm, string(k.PublicKey), string(k.PrivateKey),
		k.IsCurrent, k.CreatedAt, toNullTime(k.RotatedAt), k.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert signing key: %w", err)
	}
	return nil
}

func scanSigningKey(row pgx.Row) (*domain.SigningKey, error) {
	var k domain.SigningKey
	var pub, priv string
	var rotatedAt sql.NullTime
	if err := row.Scan(
		&k.ID, &k.KeyID, &k.Algorithm, &pub, &priv, &k.IsCurrent, &k.CreatedAt, &rotatedAt,
	); err != nil {
		return nil, err
	}
	k.PublicKey = []byte(pub)
	k.PrivateKey = []byte(priv)
	if rotatedAt.Valid {
		k.RotatedAt = &rotatedAt.Time
	}
	return &k, nil
}

func (r *SigningKeyRepo) GetCurrent(ctx context.Context) (*domain.SigningKey, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, kid, algorithm, public_key, private_key, is_current, created_at, rotated_at
		 FROM signing_keys WHERE is_current = TRUE LIMIT 1`,
	)
	k, err := scanSigningKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return k, nil
}

func (r *SigningKeyRepo) List(ctx context.Context) ([]*domain.SigningKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, kid, algorithm, public_key, private_key, is_current, created_at, rotated_at
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

func (r *SigningKeyRepo) Rotate(ctx context.Context, oldID, newID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE signing_keys SET is_current = FALSE, rotated_at = NOW() WHERE id = $1`,
		oldID,
	); err != nil {
		return fmt.Errorf("deactivate old key: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE signing_keys SET is_current = TRUE WHERE id = $1`,
		newID,
	); err != nil {
		return fmt.Errorf("activate new key: %w", err)
	}

	return tx.Commit(ctx)
}
