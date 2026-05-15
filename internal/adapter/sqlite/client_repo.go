package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type ClientRepo struct {
	db *sql.DB
}

func NewClientRepo(db *sql.DB) *ClientRepo {
	return &ClientRepo{db: db}
}

const clientColumns = `id, client_id, client_secret_hash, client_secret_plain, client_name, description, logo_url, owner_user_id,
	redirect_uris, grant_types, response_types, scopes, token_endpoint_auth_method,
	min_security_level, require_email_verified, protocol_type, is_active, is_confidential,
	created_at, updated_at`

func scanClient(row interface{ Scan(dest ...any) error }) (*domain.OIDCClient, error) {
	var c domain.OIDCClient
	var id string
	var ownerUserID sql.NullString
	var redirectURIs, grantTypes, responseTypes, scopes string

	err := row.Scan(
		&id, &c.ClientID, &c.ClientSecretHash, &c.ClientSecretPlain, &c.ClientName, &c.Description, &c.LogoURL, &ownerUserID,
		&redirectURIs, &grantTypes, &responseTypes, &scopes, &c.TokenEndpointAuthMethod,
		&c.MinSecurityLevel, &c.RequireEmailVerified, &c.ProtocolType, &c.IsActive, &c.IsConfidential,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.ID = uuid.MustParse(id)
	if ownerUserID.Valid {
		uid := uuid.MustParse(ownerUserID.String)
		c.OwnerUserID = &uid
	}

	// Unmarshal JSON arrays
	_ = json.Unmarshal([]byte(redirectURIs), &c.RedirectURIs)
	_ = json.Unmarshal([]byte(grantTypes), &c.GrantTypes)
	_ = json.Unmarshal([]byte(responseTypes), &c.ResponseTypes)
	_ = json.Unmarshal([]byte(scopes), &c.Scopes)

	if c.RedirectURIs == nil {
		c.RedirectURIs = []string{}
	}
	if c.GrantTypes == nil {
		c.GrantTypes = []string{}
	}
	if c.ResponseTypes == nil {
		c.ResponseTypes = []string{}
	}
	if c.Scopes == nil {
		c.Scopes = []string{}
	}

	return &c, nil
}

func marshalStringSlice(s []string) string {
	if s == nil {
		return "[]"
	}
	data, _ := json.Marshal(s)
	return string(data)
}

func ownerUserIDToNullString(id *uuid.UUID) sql.NullString {
	if id == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: id.String(), Valid: true}
}

func (r *ClientRepo) Create(ctx context.Context, c *domain.OIDCClient) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	query := `
		INSERT INTO oidc_clients (
			id, client_id, client_secret_hash, client_secret_plain, client_name, description, logo_url, owner_user_id,
			redirect_uris, grant_types, response_types, scopes, token_endpoint_auth_method,
			min_security_level, require_email_verified, protocol_type, is_active, is_confidential,
			created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID.String(), c.ClientID, c.ClientSecretHash, c.ClientSecretPlain, c.ClientName, c.Description, c.LogoURL,
		ownerUserIDToNullString(c.OwnerUserID),
		marshalStringSlice(c.RedirectURIs), marshalStringSlice(c.GrantTypes),
		marshalStringSlice(c.ResponseTypes), marshalStringSlice(c.Scopes),
		c.TokenEndpointAuthMethod, c.MinSecurityLevel, c.RequireEmailVerified,
		c.ProtocolType, c.IsActive, c.IsConfidential,
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert client: %w", err)
	}
	return nil
}

func (r *ClientRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.OIDCClient, error) {
	query := `SELECT ` + clientColumns + ` FROM oidc_clients WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id.String())
	c, err := scanClient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *ClientRepo) GetByClientID(ctx context.Context, clientID string) (*domain.OIDCClient, error) {
	query := `SELECT ` + clientColumns + ` FROM oidc_clients WHERE client_id = ?`
	row := r.db.QueryRowContext(ctx, query, clientID)
	c, err := scanClient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *ClientRepo) List(ctx context.Context, offset, limit int) ([]*domain.OIDCClient, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oidc_clients`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + clientColumns + ` FROM oidc_clients ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []*domain.OIDCClient
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, 0, err
		}
		clients = append(clients, c)
	}
	return clients, total, rows.Err()
}

func (r *ClientRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID, offset, limit int) ([]*domain.OIDCClient, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oidc_clients WHERE owner_user_id = ?`, ownerID.String()).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + clientColumns + ` FROM oidc_clients WHERE owner_user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, ownerID.String(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []*domain.OIDCClient
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, 0, err
		}
		clients = append(clients, c)
	}
	return clients, total, rows.Err()
}

func (r *ClientRepo) Update(ctx context.Context, c *domain.OIDCClient) error {
	c.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE oidc_clients SET
			client_name = ?, description = ?, logo_url = ?,
			redirect_uris = ?, grant_types = ?, response_types = ?, scopes = ?,
			token_endpoint_auth_method = ?, min_security_level = ?, require_email_verified = ?,
			protocol_type = ?, is_active = ?, is_confidential = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		c.ClientName, c.Description, c.LogoURL,
		marshalStringSlice(c.RedirectURIs), marshalStringSlice(c.GrantTypes),
		marshalStringSlice(c.ResponseTypes), marshalStringSlice(c.Scopes),
		c.TokenEndpointAuthMethod, c.MinSecurityLevel, c.RequireEmailVerified,
		c.ProtocolType, c.IsActive, c.IsConfidential, c.UpdatedAt,
		c.ID.String(),
	)
	if err != nil {
		return err
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

func (r *ClientRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oidc_clients WHERE id = ?`, id.String())
	return err
}

func (r *ClientRepo) UpdateSecret(ctx context.Context, id uuid.UUID, hash, plain string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE oidc_clients SET client_secret_hash = ?, client_secret_plain = ?, updated_at = ? WHERE id = ?`,
		hash, plain, now, id.String(),
	)
	return err
}
