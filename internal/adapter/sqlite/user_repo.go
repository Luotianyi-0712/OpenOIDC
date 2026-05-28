package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const userSelectColumns = `id, uid, email, email_verified, password_hash, display_name, alias,
	avatar_url, security_level, role, status, risk_report_email_enabled, last_login_at, created_at, updated_at`

func normalizeUserDefaults(u *domain.User) {
	if strings.TrimSpace(u.Role) == "" {
		u.Role = domain.RoleUser
	}
	if u.Status == "" {
		u.Status = domain.UserStatusActive
	}
}

func scanUser(row interface{ Scan(dest ...any) error }) (*domain.User, error) {
	var u domain.User
	var id string
	var alias sql.NullString
	var lastLogin sql.NullTime
	var status string

	err := row.Scan(
		&id, &u.UID, &u.Email, &u.EmailVerified, &u.PasswordHash, &u.DisplayName, &alias,
		&u.AvatarURL, &u.SecurityLevel, &u.Role, &status, &u.RiskReportEmailEnabled, &lastLogin,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.ID = uuid.MustParse(id)
	u.Alias = fromNullString(alias)
	u.LastLoginAt = fromNullTime(lastLogin)
	u.Status = domain.UserStatus(status)
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	normalizeUserDefaults(u)
	u.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin user create: %w", err)
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, `UPDATE user_uid_sequence SET next_uid = next_uid + 1 WHERE id = 1 RETURNING next_uid - 1`).Scan(&u.UID); err != nil {
		return fmt.Errorf("allocate user uid: %w", err)
	}

	query := `
		INSERT INTO users (
			id, uid, email, email_verified, password_hash, display_name, alias,
			avatar_url, security_level, role, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	if _, err := tx.ExecContext(ctx, query,
		u.ID.String(), u.UID, u.Email, u.EmailVerified, u.PasswordHash, u.DisplayName, toNullString(u.Alias),
		u.AvatarURL, u.SecurityLevel, u.Role, string(u.Status), u.CreatedAt, u.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user create: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = ? AND deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, id.String())
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByUID(ctx context.Context, uid int64) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE uid = ? AND deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, uid)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE email = ? AND deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByAlias(ctx context.Context, alias string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE alias = ? AND deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, alias)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	normalizeUserDefaults(u)
	u.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE users SET
			email = ?, email_verified = ?, display_name = ?, alias = ?,
			avatar_url = ?, security_level = ?, role = ?, status = ?,
			risk_report_email_enabled = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		u.Email, u.EmailVerified, u.DisplayName, toNullString(u.Alias),
		u.AvatarURL, u.SecurityLevel, u.Role, string(u.Status), u.RiskReportEmailEnabled, u.UpdatedAt,
		u.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
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

func (r *UserRepo) UpdateSecurityLevel(ctx context.Context, id uuid.UUID, level int) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET security_level = ?, updated_at = ? WHERE id = ?`,
		level, now, id.String(),
	)
	return err
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`,
		now, now, id.String(),
	)
	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		hash, now, id.String(),
	)
	return err
}

func (r *UserRepo) List(ctx context.Context, opts port.ListUsersOptions) ([]*domain.User, int64, error) {
	args := []any{}
	where := []string{"deleted_at IS NULL"}

	if opts.Search != "" {
		pattern := "%" + opts.Search + "%"
		args = append(args, pattern, pattern, pattern, pattern)
		where = append(where, "(id LIKE ? OR CAST(uid AS TEXT) LIKE ? OR email LIKE ? OR display_name LIKE ?)")
	}
	if opts.Status != nil {
		args = append(args, string(*opts.Status))
		where = append(where, "status = ?")
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	countQuery := "SELECT COUNT(*) FROM users " + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	listArgs := append(args, limit, offset)
	listQuery := fmt.Sprintf(
		"SELECT %s FROM users %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		userSelectColumns, whereClause,
	)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *UserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = 'deleted', deleted_at = ?, updated_at = ? WHERE id = ?`,
		now, now, id.String(),
	)
	return err
}
