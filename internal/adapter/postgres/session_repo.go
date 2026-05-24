package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	db *pgxpool.Pool
}

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{db: db}
}

func scanSession(row pgx.Row) (*domain.UserSession, error) {
	var s domain.UserSession
	var ip sql.NullString
	var ua sql.NullString
	if err := row.Scan(&s.ID, &s.UserID, &s.SessionToken, &ip, &ua, &s.ExpiresAt, &s.CreatedAt); err != nil {
		return nil, err
	}
	if ip.Valid {
		s.IPAddress = &ip.String
	}
	if ua.Valid {
		s.UserAgent = &ua.String
	}
	return &s, nil
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.UserSession) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	var ip, ua string
	if s.IPAddress != nil {
		ip = *s.IPAddress
	}
	if s.UserAgent != nil {
		ua = *s.UserAgent
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_sessions (id, user_id, token, ip_address, user_agent, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.UserID, s.SessionToken, ip, ua, s.ExpiresAt, s.CreatedAt,
	)
	return err
}

func (r *SessionRepo) GetByToken(ctx context.Context, token string) (*domain.UserSession, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, token, ip_address, user_agent, expires_at, created_at
		 FROM user_sessions WHERE token = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		token,
	)
	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *SessionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, token, ip_address, user_agent, expires_at, created_at
		 FROM user_sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID,
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
	_, err := r.db.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_sessions WHERE expires_at < NOW() OR revoked_at IS NOT NULL`,
	)
	return err
}

func (r *SessionRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func (r *SessionRepo) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_sessions WHERE revoked_at IS NULL AND expires_at > NOW()`,
	).Scan(&count)
	return count, err
}
