package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func scanSession(row interface{ Scan(dest ...any) error }) (*domain.UserSession, error) {
	var s domain.UserSession
	var id, userID string
	var ip, ua sql.NullString

	if err := row.Scan(&id, &userID, &s.SessionToken, &ip, &ua, &s.ExpiresAt, &s.CreatedAt); err != nil {
		return nil, err
	}
	s.ID = uuid.MustParse(id)
	s.UserID = uuid.MustParse(userID)
	s.IPAddress = fromNullString(ip)
	s.UserAgent = fromNullString(ua)
	return &s, nil
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.UserSession) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_sessions (id, user_id, session_token, ip_address, user_agent, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID.String(), s.UserID.String(), s.SessionToken,
		toNullString(s.IPAddress), toNullString(s.UserAgent),
		s.ExpiresAt, s.CreatedAt,
	)
	return err
}

func (r *SessionRepo) GetByToken(ctx context.Context, token string) (*domain.UserSession, error) {
	now := time.Now().UTC()
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, session_token, ip_address, user_agent, expires_at, created_at
		 FROM user_sessions WHERE session_token = ? AND revoked_at IS NULL AND expires_at > ?`,
		token, now,
	)
	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *SessionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserSession, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, session_token, ip_address, user_agent, expires_at, created_at
		 FROM user_sessions WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.UserSession
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_sessions SET revoked_at = ? WHERE id = ?`,
		now, id.String(),
	)
	return err
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_sessions WHERE expires_at < ? OR revoked_at IS NOT NULL`,
		now,
	)
	return err
}

func (r *SessionRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		now, userID.String(),
	)
	return err
}

func (r *SessionRepo) CountActive(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_sessions WHERE revoked_at IS NULL AND expires_at > ?`,
		now,
	).Scan(&count)
	return count, err
}
