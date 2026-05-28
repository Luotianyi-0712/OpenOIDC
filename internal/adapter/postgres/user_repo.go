package postgres

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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
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

	query := `
		INSERT INTO users (
			id, email, email_verified, password_hash, display_name, alias,
			avatar_url, security_level, role, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING uid
	`
	var alias sql.NullString
	if u.Alias != nil {
		alias = sql.NullString{String: *u.Alias, Valid: true}
	}
	if err := r.db.QueryRow(ctx, query,
		u.ID, u.Email, u.EmailVerified, u.PasswordHash, u.DisplayName, alias,
		u.AvatarURL, u.SecurityLevel, u.Role, string(u.Status), u.CreatedAt, u.UpdatedAt,
	).Scan(&u.UID); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func normalizeUserDefaults(u *domain.User) {
	if strings.TrimSpace(u.Role) == "" {
		u.Role = domain.RoleUser
	}
	if u.Status == "" {
		u.Status = domain.UserStatusActive
	}
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var alias sql.NullString
	var lastLogin sql.NullTime
	var status string
	err := row.Scan(
		&u.ID, &u.UID, &u.Email, &u.EmailVerified, &u.PasswordHash, &u.DisplayName, &alias,
		&u.AvatarURL, &u.SecurityLevel, &u.Role, &status, &u.RiskReportEmailEnabled, &lastLogin,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if alias.Valid {
		u.Alias = &alias.String
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	u.Status = domain.UserStatus(status)
	return &u, nil
}

const userSelectColumns = `id, uid, email, email_verified, password_hash, display_name, alias,
	avatar_url, security_level, role, status, risk_report_email_enabled, last_login_at, created_at, updated_at`

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`
	row := r.db.QueryRow(ctx, query, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByUID(ctx context.Context, uid int64) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE uid = $1 AND deleted_at IS NULL`
	row := r.db.QueryRow(ctx, query, uid)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE email = $1 AND deleted_at IS NULL`
	row := r.db.QueryRow(ctx, query, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByAlias(ctx context.Context, alias string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE alias = $1 AND deleted_at IS NULL`
	row := r.db.QueryRow(ctx, query, alias)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	normalizeUserDefaults(u)
	u.UpdatedAt = time.Now().UTC()
	var alias sql.NullString
	if u.Alias != nil {
		alias = sql.NullString{String: *u.Alias, Valid: true}
	}
	query := `
		UPDATE users SET
			email = $2, email_verified = $3, display_name = $4, alias = $5,
			avatar_url = $6, security_level = $7, role = $8, status = $9,
			risk_report_email_enabled = $10, updated_at = $11
		WHERE id = $1
	`
	tag, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.EmailVerified, u.DisplayName, alias,
		u.AvatarURL, u.SecurityLevel, u.Role, string(u.Status), u.RiskReportEmailEnabled, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *UserRepo) UpdateSecurityLevel(ctx context.Context, id uuid.UUID, level int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET security_level = $2, updated_at = NOW() WHERE id = $1`,
		id, level,
	)
	return err
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		id, hash,
	)
	return err
}

func (r *UserRepo) List(ctx context.Context, opts port.ListUsersOptions) ([]*domain.User, int64, error) {
	args := []any{}
	where := []string{"deleted_at IS NULL"}

	if opts.Search != "" {
		args = append(args, "%"+opts.Search+"%")
		where = append(where, fmt.Sprintf("(id::text ILIKE $%d OR uid::text ILIKE $%d OR email ILIKE $%d OR display_name ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	if opts.Status != nil {
		args = append(args, string(*opts.Status))
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	countQuery := "SELECT COUNT(*) FROM users " + whereClause
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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
	args = append(args, limit, offset)
	listQuery := fmt.Sprintf(
		"SELECT %s FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		userSelectColumns, whereClause, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, listQuery, args...)
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
	_, err := r.db.Exec(ctx,
		`UPDATE users SET status = 'deleted', deleted_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}
