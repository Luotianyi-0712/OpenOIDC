package postgres

import (
	"context"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClientAccessRuleRepo struct {
	db *pgxpool.Pool
}

func NewClientAccessRuleRepo(db *pgxpool.Pool) *ClientAccessRuleRepo {
	return &ClientAccessRuleRepo{db: db}
}

func (r *ClientAccessRuleRepo) Create(ctx context.Context, rule *domain.ClientAccessRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO client_access_rules (id, client_id, rule_type, rule_value, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		rule.ID, rule.ClientID, rule.RuleType, rule.Value, rule.CreatedAt,
	)
	return err
}

func (r *ClientAccessRuleRepo) ListByClient(ctx context.Context, clientID uuid.UUID) ([]*domain.ClientAccessRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, rule_type, rule_value, created_at FROM client_access_rules
		 WHERE client_id = $1 ORDER BY priority DESC, created_at`,
		clientID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.ClientAccessRule
	for rows.Next() {
		var rule domain.ClientAccessRule
		if err := rows.Scan(&rule.ID, &rule.ClientID, &rule.RuleType, &rule.Value, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (r *ClientAccessRuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM client_access_rules WHERE id = $1`, id)
	return err
}
