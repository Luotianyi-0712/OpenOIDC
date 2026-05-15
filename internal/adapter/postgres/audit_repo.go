package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepo struct {
	db *pgxpool.Pool
}

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) CreateLog(ctx context.Context, log *domain.AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	var details []byte
	if log.Details != nil {
		var err error
		details, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("marshal details: %w", err)
		}
	}

	var resourceType, resourceID, ip, ua string
	if log.ResourceType != nil {
		resourceType = *log.ResourceType
	}
	if log.ResourceID != nil {
		resourceID = *log.ResourceID
	}
	if log.IPAddress != nil {
		ip = *log.IPAddress
	}
	if log.UserAgent != nil {
		ua = *log.UserAgent
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO audit_log
		 (id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		log.ID, log.UserID, log.Action, resourceType, resourceID, ip, ua, details, log.CreatedAt,
	)
	return err
}

func (r *AuditRepo) ListLogs(ctx context.Context, opts port.ListAuditOptions) ([]*domain.AuditLog, int64, error) {
	args := []any{}
	where := []string{"1=1"}

	if opts.UserID != nil {
		args = append(args, *opts.UserID)
		where = append(where, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if opts.Action != "" {
		args = append(args, opts.Action)
		where = append(where, fmt.Sprintf("action = $%d", len(args)))
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	query := fmt.Sprintf(
		`SELECT id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at
		 FROM audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var userID *uuid.UUID
		var resourceType, resourceID, ip, ua string
		var details []byte
		if err := rows.Scan(
			&l.ID, &userID, &l.Action, &resourceType, &resourceID, &ip, &ua, &details, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		l.UserID = userID
		if resourceType != "" {
			l.ResourceType = &resourceType
		}
		if resourceID != "" {
			l.ResourceID = &resourceID
		}
		if ip != "" {
			l.IPAddress = &ip
		}
		if ua != "" {
			l.UserAgent = &ua
		}
		if len(details) > 0 {
			_ = json.Unmarshal(details, &l.Details)
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}

func (r *AuditRepo) CreateSecurityLevelChange(ctx context.Context, c *domain.SecurityLevelChange) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	var ruleID *uuid.UUID
	if c.MatchedRuleID != nil {
		ruleID = c.MatchedRuleID
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO security_level_changes
		 (id, user_id, old_level, new_level, reason, rule_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.UserID, c.OldLevel, c.NewLevel, c.Reason, ruleID, c.CreatedAt,
	)
	return err
}

func (r *AuditRepo) ListSecurityLevelChanges(ctx context.Context, userID uuid.UUID) ([]*domain.SecurityLevelChange, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, old_level, new_level, reason, rule_id, created_at
		 FROM security_level_changes WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []*domain.SecurityLevelChange
	for rows.Next() {
		var c domain.SecurityLevelChange
		var ruleID *uuid.UUID
		if err := rows.Scan(&c.ID, &c.UserID, &c.OldLevel, &c.NewLevel, &c.Reason, &ruleID, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.MatchedRuleID = ruleID
		changes = append(changes, &c)
	}
	return changes, rows.Err()
}
