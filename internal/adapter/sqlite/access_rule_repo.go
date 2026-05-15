package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
)

type ClientAccessRuleRepo struct {
	db *sql.DB
}

func NewClientAccessRuleRepo(db *sql.DB) *ClientAccessRuleRepo {
	return &ClientAccessRuleRepo{db: db}
}

func (r *ClientAccessRuleRepo) Create(ctx context.Context, rule *domain.ClientAccessRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO client_access_rules (id, client_id, rule_type, value, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		rule.ID.String(), rule.ClientID.String(), rule.RuleType, rule.Value, rule.CreatedAt,
	)
	return err
}

func (r *ClientAccessRuleRepo) ListByClient(ctx context.Context, clientID uuid.UUID) ([]*domain.ClientAccessRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, client_id, rule_type, value, created_at FROM client_access_rules
		 WHERE client_id = ? ORDER BY created_at`,
		clientID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.ClientAccessRule
	for rows.Next() {
		var rule domain.ClientAccessRule
		var id, cid string
		if err := rows.Scan(&id, &cid, &rule.RuleType, &rule.Value, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rule.ID = uuid.MustParse(id)
		rule.ClientID = uuid.MustParse(cid)
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

func (r *ClientAccessRuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM client_access_rules WHERE id = ?`, id.String())
	return err
}
