package postgres

import (
	"context"
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

type RuleRepo struct {
	db *pgxpool.Pool
}

func NewRuleRepo(db *pgxpool.Pool) *RuleRepo {
	return &RuleRepo{db: db}
}

const ruleColumns = `id, name, description, level, priority, conditions, is_active, created_at, updated_at`

func scanRule(row pgx.Row) (*domain.SecurityLevelRule, error) {
	var r domain.SecurityLevelRule
	var conditions []byte
	err := row.Scan(
		&r.ID, &r.Name, &r.Description, &r.Level, &r.Priority,
		&conditions, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(conditions) > 0 {
		if err := json.Unmarshal(conditions, &r.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}
	}
	return &r, nil
}

func (r *RuleRepo) Create(ctx context.Context, rule *domain.SecurityLevelRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	now := time.Now().UTC()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	conditions, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO security_level_rules
		 (id, name, description, level, priority, conditions, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rule.ID, rule.Name, rule.Description, rule.Level, rule.Priority,
		conditions, rule.IsActive, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

func (r *RuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SecurityLevelRule, error) {
	row := r.db.QueryRow(ctx, `SELECT `+ruleColumns+` FROM security_level_rules WHERE id = $1`, id)
	rule, err := scanRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

func (r *RuleRepo) ListActive(ctx context.Context) ([]*domain.SecurityLevelRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+ruleColumns+` FROM security_level_rules WHERE is_active = TRUE
		 ORDER BY level DESC, priority DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.SecurityLevelRule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *RuleRepo) ListAll(ctx context.Context) ([]*domain.SecurityLevelRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+ruleColumns+` FROM security_level_rules ORDER BY level DESC, priority DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.SecurityLevelRule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *RuleRepo) Update(ctx context.Context, rule *domain.SecurityLevelRule) error {
	rule.UpdatedAt = time.Now().UTC()
	conditions, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE security_level_rules SET
		 name = $2, description = $3, level = $4, priority = $5, conditions = $6,
		 is_active = $7, updated_at = $8 WHERE id = $1`,
		rule.ID, rule.Name, rule.Description, rule.Level, rule.Priority,
		conditions, rule.IsActive, rule.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *RuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM security_level_rules WHERE id = $1`, id)
	return err
}
