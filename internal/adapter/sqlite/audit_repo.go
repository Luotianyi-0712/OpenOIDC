package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

// AuditRepo implements port.AuditRepository using SQLite.
type AuditRepo struct {
	db *sql.DB
}

// NewAuditRepo returns a new AuditRepo.
func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// CreateLog inserts a new audit log entry.
func (r *AuditRepo) CreateLog(ctx context.Context, log *domain.AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	var details sql.NullString
	if log.Details != nil {
		b, err := json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("marshal details: %w", err)
		}
		details = sql.NullString{String: string(b), Valid: true}
	}

	var userID sql.NullString
	if log.UserID != nil {
		userID = sql.NullString{String: log.UserID.String(), Valid: true}
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log
		 (id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID.String(), userID, log.Action,
		toNullString(log.ResourceType), toNullString(log.ResourceID),
		toNullString(log.IPAddress), toNullString(log.UserAgent),
		details, log.CreatedAt,
	)
	return err
}

// ListLogs returns a paginated list of audit logs with optional filters.
func (r *AuditRepo) ListLogs(ctx context.Context, opts port.ListAuditOptions) ([]*domain.AuditLog, int64, error) {
	var args []any
	var where []string

	if opts.UserID != nil {
		args = append(args, opts.UserID.String())
		where = append(where, fmt.Sprintf("user_id = ?"))
	}
	if opts.Action != "" {
		args = append(args, opts.Action)
		where = append(where, fmt.Sprintf("action = ?"))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM audit_log " + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
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

	selectArgs := make([]any, len(args))
	copy(selectArgs, args)
	selectArgs = append(selectArgs, limit, offset)

	query := fmt.Sprintf(
		`SELECT id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at
		 FROM audit_log %s ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		whereClause,
	)

	rows, err := r.db.QueryContext(ctx, query, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var id string
		var userID sql.NullString
		var resourceType, resourceID, ip, ua sql.NullString
		var details sql.NullString

		if err := rows.Scan(
			&id, &userID, &l.Action, &resourceType, &resourceID,
			&ip, &ua, &details, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		l.ID = uuid.MustParse(id)
		if userID.Valid {
			uid := uuid.MustParse(userID.String)
			l.UserID = &uid
		}
		l.ResourceType = fromNullString(resourceType)
		l.ResourceID = fromNullString(resourceID)
		l.IPAddress = fromNullString(ip)
		l.UserAgent = fromNullString(ua)

		if details.Valid && details.String != "" {
			_ = json.Unmarshal([]byte(details.String), &l.Details)
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}

// CreateSecurityLevelChange inserts a new security level change record.
func (r *AuditRepo) CreateSecurityLevelChange(ctx context.Context, c *domain.SecurityLevelChange) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}

	var ruleID sql.NullString
	if c.MatchedRuleID != nil {
		ruleID = sql.NullString{String: c.MatchedRuleID.String(), Valid: true}
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO security_level_changes
		 (id, user_id, old_level, new_level, reason, matched_rule_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID.String(), c.UserID.String(), c.OldLevel, c.NewLevel, c.Reason, ruleID, c.CreatedAt,
	)
	return err
}

// ListSecurityLevelChanges returns all security level changes for a user.
func (r *AuditRepo) ListSecurityLevelChanges(ctx context.Context, userID uuid.UUID) ([]*domain.SecurityLevelChange, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, old_level, new_level, reason, matched_rule_id, created_at
		 FROM security_level_changes WHERE user_id = ? ORDER BY created_at DESC`,
		userID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []*domain.SecurityLevelChange
	for rows.Next() {
		var c domain.SecurityLevelChange
		var id, uid string
		var ruleID sql.NullString

		if err := rows.Scan(&id, &uid, &c.OldLevel, &c.NewLevel, &c.Reason, &ruleID, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ID = uuid.MustParse(id)
		c.UserID = uuid.MustParse(uid)
		if ruleID.Valid {
			rid := uuid.MustParse(ruleID.String)
			c.MatchedRuleID = &rid
		}
		changes = append(changes, &c)
	}
	return changes, rows.Err()
}
