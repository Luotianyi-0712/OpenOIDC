package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
)

// AliasRestrictionRepo implements port.AliasRestrictionRepository using SQLite.
type AliasRestrictionRepo struct {
	db *sql.DB
}

// NewAliasRestrictionRepo returns a new AliasRestrictionRepo.
func NewAliasRestrictionRepo(db *sql.DB) *AliasRestrictionRepo {
	return &AliasRestrictionRepo{db: db}
}

// Create inserts a new alias restriction.
func (r *AliasRestrictionRepo) Create(ctx context.Context, restriction *domain.AliasRestriction) error {
	if restriction.ID == uuid.Nil {
		restriction.ID = uuid.New()
	}
	if restriction.CreatedAt.IsZero() {
		restriction.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO alias_restrictions (id, pattern, restriction_type, reason, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		restriction.ID.String(), restriction.Pattern, restriction.RestrictionType,
		restriction.Reason, restriction.CreatedAt,
	)
	return err
}

// List returns all alias restrictions ordered by created_at descending.
func (r *AliasRestrictionRepo) List(ctx context.Context) ([]*domain.AliasRestriction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, pattern, restriction_type, reason, created_at
		 FROM alias_restrictions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var restrictions []*domain.AliasRestriction
	for rows.Next() {
		var ar domain.AliasRestriction
		var id string
		if err := rows.Scan(&id, &ar.Pattern, &ar.RestrictionType, &ar.Reason, &ar.CreatedAt); err != nil {
			return nil, err
		}
		ar.ID = uuid.MustParse(id)
		restrictions = append(restrictions, &ar)
	}
	return restrictions, rows.Err()
}

// Delete removes an alias restriction by ID.
func (r *AliasRestrictionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM alias_restrictions WHERE id = ?`,
		id.String(),
	)
	return err
}
