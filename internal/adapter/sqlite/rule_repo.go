package sqlite

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
)

type RuleRepo struct {
	db *sql.DB
}

func NewRuleRepo(db *sql.DB) *RuleRepo {
	return &RuleRepo{db: db}
}

const ruleColumns = `id, name, description, level, priority, conditions, is_active, created_at, updated_at`

func scanRule(row interface{ Scan(dest ...any) error }) (*domain.SecurityLevelRule, error) {
	var r domain.SecurityLevelRule
	var id string
	var conditions string

	err := row.Scan(
		&id, &r.Name, &r.Description, &r.Level, &r.Priority,
		&conditions, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	r.ID = uuid.MustParse(id)
	if conditions != "" {
		if err := json.Unmarshal([]byte(conditions), &r.Conditions); err != nil {
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

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO security_level_rules
		 (id, name, description, level, priority, conditions, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID.String(), rule.Name, rule.Description, rule.Level, rule.Priority,
		string(conditions), rule.IsActive, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

func (r *RuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SecurityLevelRule, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+ruleColumns+` FROM security_level_rules WHERE id = ?`, id.String())
	rule, err := scanRule(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

func (r *RuleRepo) ListActive(ctx context.Context) ([]*domain.SecurityLevelRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+ruleColumns+` FROM security_level_rules WHERE is_active = 1
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
	rows, err := r.db.QueryContext(ctx,
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
	res, err := r.db.ExecContext(ctx,
		`UPDATE security_level_rules SET
		 name = ?, description = ?, level = ?, priority = ?, conditions = ?,
		 is_active = ?, updated_at = ? WHERE id = ?`,
		rule.Name, rule.Description, rule.Level, rule.Priority,
		string(conditions), rule.IsActive, rule.UpdatedAt, rule.ID.String(),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *RuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM security_level_rules WHERE id = ?`, id.String())
	return err
}
