package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anthropic/oidc-platform/internal/domain"
)

type ConsentRepo struct {
	db *pgxpool.Pool
}

func NewConsentRepo(db *pgxpool.Pool) *ConsentRepo {
	return &ConsentRepo{db: db}
}

func (r *ConsentRepo) ListAuthorizedApps(ctx context.Context, userID uuid.UUID) ([]*domain.UserAuthorization, error) {
	// Postgres uses oauth2_access_tokens table (not oauth2_sessions).
	// The 'subject' column stores the user ID.
	rows, err := r.db.Query(ctx,
		`SELECT oa.client_id, oc.name, MIN(oa.created_at) as granted_at
		 FROM oauth2_access_tokens oa
		 JOIN oidc_clients oc ON oc.client_id = oa.client_id
		 WHERE oa.active = true
		   AND oa.subject = $1
		   AND (oa.expires_at IS NULL OR oa.expires_at > NOW())
		 GROUP BY oa.client_id, oc.name
		 ORDER BY granted_at DESC`,
		userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.UserAuthorization
	for rows.Next() {
		var clientID, clientName string
		var grantedAt time.Time

		if err := rows.Scan(&clientID, &clientName, &grantedAt); err != nil {
			continue
		}

		auth := &domain.UserAuthorization{
			ID:         uuid.New(),
			UserID:     userID,
			ClientID:   clientID,
			ClientName: clientName,
			GrantedAt:  grantedAt,
			LastUsedAt: grantedAt,
		}
		results = append(results, auth)
	}

	return results, nil
}

func (r *ConsentRepo) CountUniqueUsers(ctx context.Context, clientID string) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT subject) FROM oauth2_access_tokens
		 WHERE client_id = $1 AND active = true
		   AND subject != ''
		   AND (expires_at IS NULL OR expires_at > NOW())`,
		clientID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteByUserAndClient revokes a user's authorization to a specific client.
func (r *ConsentRepo) DeleteByUserAndClient(ctx context.Context, userID uuid.UUID, clientID string) error {
	// Deactivate across all token tables.
	uid := userID.String()
	_, _ = r.db.Exec(ctx, `UPDATE oauth2_access_tokens SET active = false WHERE subject = $1 AND client_id = $2`, uid, clientID)
	_, _ = r.db.Exec(ctx, `UPDATE oauth2_refresh_tokens SET active = false WHERE subject = $1 AND client_id = $2`, uid, clientID)
	_, _ = r.db.Exec(ctx, `UPDATE oauth2_authorization_codes SET active = false WHERE subject = $1 AND client_id = $2`, uid, clientID)
	return nil
}
