package postgres

import (
	"context"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AliasRestrictionRepo struct {
	db *pgxpool.Pool
}

func NewAliasRestrictionRepo(db *pgxpool.Pool) *AliasRestrictionRepo {
	return &AliasRestrictionRepo{db: db}
}

func (r *AliasRestrictionRepo) Create(ctx context.Context, restriction *domain.AliasRestriction) error {
	if restriction.ID == uuid.Nil {
		restriction.ID = uuid.New()
	}
	if restriction.CreatedAt.IsZero() {
		restriction.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO alias_restrictions (id, pattern, restriction_type, description, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		restriction.ID, restriction.Pattern, restriction.RestrictionType, restriction.Reason, restriction.CreatedAt,
	)
	return err
}

func (r *AliasRestrictionRepo) List(ctx context.Context) ([]*domain.AliasRestriction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, pattern, restriction_type, description, created_at FROM alias_restrictions ORDER BY pattern`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var restrictions []*domain.AliasRestriction
	for rows.Next() {
		var ar domain.AliasRestriction
		if err := rows.Scan(&ar.ID, &ar.Pattern, &ar.RestrictionType, &ar.Reason, &ar.CreatedAt); err != nil {
			return nil, err
		}
		restrictions = append(restrictions, &ar)
	}
	return restrictions, rows.Err()
}

func (r *AliasRestrictionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM alias_restrictions WHERE id = $1`, id)
	return err
}
