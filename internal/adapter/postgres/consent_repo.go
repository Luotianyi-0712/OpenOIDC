package postgres

import (
	"context"
	"strings"
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

func (r *ConsentRepo) ListClientUsers(ctx context.Context, client *domain.OIDCClient, search string, offset, limit int) ([]*domain.DeveloperAppUserSummary, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	search = strings.TrimSpace(search)
	pattern := "%" + search + "%"

	where := `oa.client_id = $1 AND oa.active = true
		AND oa.subject != ''
		AND (oa.expires_at IS NULL OR oa.expires_at > NOW())
		AND u.deleted_at IS NULL
		AND ($2 = '' OR u.id::text ILIKE $3 OR u.email ILIKE $3 OR u.display_name ILIKE $3)`

	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT u.id)
		 FROM oauth2_access_tokens oa
		 JOIN users u ON u.id::text = oa.subject
		 WHERE `+where,
		client.ClientID, search, pattern,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT u.id, u.display_name, u.email, u.security_level,
		        COALESCE(array_remove(array_agg(DISTINCT sb.provider), NULL), ARRAY[]::text[]) AS providers,
		        BOOL_OR(car.id IS NOT NULL) AS blocked,
		        MIN(oa.created_at) AS granted_at,
		        MAX(oa.created_at) AS last_used_at
		 FROM oauth2_access_tokens oa
		 JOIN users u ON u.id::text = oa.subject
		 LEFT JOIN social_bindings sb ON sb.user_id = u.id AND sb.status = 'active'
		 LEFT JOIN client_access_rules car ON car.client_id = $4 AND car.rule_type = $5 AND car.rule_value = u.id::text
		 WHERE `+where+`
		 GROUP BY u.id, u.display_name, u.email, u.security_level
		 ORDER BY last_used_at DESC
		 LIMIT $6 OFFSET $7`,
		client.ClientID, search, pattern, client.ID, string(domain.AccessRuleUserDeny), limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]*domain.DeveloperAppUserSummary, 0)
	for rows.Next() {
		item := &domain.DeveloperAppUserSummary{}
		if err := rows.Scan(&item.UID, &item.DisplayName, &item.Email, &item.SecurityLevel, &item.Providers, &item.Blocked, &item.GrantedAt, &item.LastUsedAt); err != nil {
			return nil, 0, err
		}
		if item.Providers == nil {
			item.Providers = []string{}
		}
		users = append(users, item)
	}
	return users, total, rows.Err()
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
