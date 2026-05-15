package sqlite

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
)

type ConsentRepo struct {
	db *sql.DB
}

func NewConsentRepo(db *sql.DB) *ConsentRepo {
	return &ConsentRepo{db: db}
}

// ListAuthorizedApps returns a deduplicated list of apps a user has authorized.
// Uses the subject column in oauth2_sessions to filter by user.
func (r *ConsentRepo) ListAuthorizedApps(ctx context.Context, userID uuid.UUID) ([]*domain.UserAuthorization, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT os.client_id, oc.client_name, MIN(os.created_at) as granted_at
		 FROM oauth2_sessions os
		 JOIN oidc_clients oc ON oc.client_id = os.client_id
		 WHERE os.active = 1 AND os.session_type = 'access_token'
		   AND os.subject = ?
		   AND (os.expires_at IS NULL OR os.expires_at > datetime('now'))
		 GROUP BY os.client_id, oc.client_name
		 ORDER BY granted_at DESC`,
		userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.UserAuthorization
	for rows.Next() {
		var clientID, clientName string
		var createdAt sql.NullTime

		if err := rows.Scan(&clientID, &clientName, &createdAt); err != nil {
			continue
		}

		auth := &domain.UserAuthorization{
			ID:         uuid.New(),
			UserID:     userID,
			ClientID:   clientID,
			ClientName: clientName,
		}
		if createdAt.Valid {
			auth.GrantedAt = createdAt.Time
			auth.LastUsedAt = createdAt.Time
		}
		results = append(results, auth)
	}

	return results, nil
}

func (r *ConsentRepo) CountUniqueUsers(ctx context.Context, clientID string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT subject) FROM oauth2_sessions
		 WHERE client_id = ? AND active = 1 AND session_type = 'access_token'
		   AND subject != ''
		   AND (expires_at IS NULL OR expires_at > datetime('now'))`,
		clientID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteByUserAndClient removes all sessions for a user+client pair (revoke authorization).
func (r *ConsentRepo) DeleteByUserAndClient(ctx context.Context, userID uuid.UUID, clientID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE oauth2_sessions SET active = 0 WHERE subject = ? AND client_id = ?`,
		userID.String(), clientID)
	return err
}
