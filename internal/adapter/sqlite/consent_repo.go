package sqlite

import (
	"context"
	"database/sql"
	"strings"

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
			 WHERE os.active = 1 AND os.session_type IN ('access_token', 'refresh_token')
			   AND os.subject = ?
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
		var createdAt sql.NullString

		if err := rows.Scan(&clientID, &clientName, &createdAt); err != nil {
			continue
		}

		auth := &domain.UserAuthorization{
			ID:         uuid.New(),
			UserID:     userID,
			ClientID:   clientID,
			ClientName: clientName,
		}
		if createdAt.Valid && createdAt.String != "" {
			grantedAt := parseTimeLoose(createdAt.String)
			auth.GrantedAt = grantedAt
			auth.LastUsedAt = grantedAt
		}
		results = append(results, auth)
	}

	return results, nil
}

func (r *ConsentRepo) CountUniqueUsers(ctx context.Context, clientID string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT subject) FROM oauth2_sessions
			 WHERE client_id = ? AND active = 1 AND session_type IN ('access_token', 'refresh_token')
			   AND subject != ''`,
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

	where := `os.client_id = ? AND os.active = 1 AND os.session_type IN ('access_token', 'refresh_token')
		AND os.subject != ''
		AND u.deleted_at IS NULL
		AND (? = '' OR u.id LIKE ? OR CAST(u.uid AS TEXT) LIKE ? OR u.email LIKE ? OR u.display_name LIKE ?)`

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT u.id)
			 FROM oauth2_sessions os
			 JOIN users u ON u.id = os.subject
			 WHERE `+where,
		client.ClientID, search, pattern, pattern, pattern, pattern,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.uid, u.display_name, u.email, u.security_level,
		        COALESCE(GROUP_CONCAT(DISTINCT sb.provider), '') AS providers,
		        MAX(CASE WHEN car.id IS NULL THEN 0 ELSE 1 END) AS blocked,
		        MIN(os.created_at) AS granted_at,
		        MAX(os.created_at) AS last_used_at
		 FROM oauth2_sessions os
		 JOIN users u ON u.id = os.subject
		 LEFT JOIN social_bindings sb ON sb.user_id = u.id AND sb.status = 'active'
		 LEFT JOIN client_access_rules car ON car.client_id = ? AND car.rule_type = ? AND car.value = u.id
		 WHERE `+where+`
		 GROUP BY u.id, u.uid, u.display_name, u.email, u.security_level
		 ORDER BY last_used_at DESC
		 LIMIT ? OFFSET ?`,
		client.ID.String(), string(domain.AccessRuleUserDeny), client.ClientID, search, pattern, pattern, pattern, pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]*domain.DeveloperAppUserSummary, 0)
	for rows.Next() {
		var id string
		var providers string
		var blocked int
		var grantedAt, lastUsedAt sql.NullString
		item := &domain.DeveloperAppUserSummary{}
		if err := rows.Scan(&id, &item.UID, &item.DisplayName, &item.Email, &item.SecurityLevel, &providers, &blocked, &grantedAt, &lastUsedAt); err != nil {
			return nil, 0, err
		}
		parsedID, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		item.ID = parsedID
		if providers != "" {
			item.Providers = strings.Split(providers, ",")
		} else {
			item.Providers = []string{}
		}
		item.Blocked = blocked > 0
		if grantedAt.Valid && grantedAt.String != "" {
			item.GrantedAt = parseTimeLoose(grantedAt.String)
		}
		if lastUsedAt.Valid && lastUsedAt.String != "" {
			item.LastUsedAt = parseTimeLoose(lastUsedAt.String)
		}
		users = append(users, item)
	}
	return users, total, rows.Err()
}

// DeleteByUserAndClient removes all sessions for a user+client pair (revoke authorization).
func (r *ConsentRepo) DeleteByUserAndClient(ctx context.Context, userID uuid.UUID, clientID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE oauth2_sessions SET active = 0 WHERE subject = ? AND client_id = ?`,
		userID.String(), clientID)
	return err
}
