package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClientRepo struct {
	db *pgxpool.Pool
}

func NewClientRepo(db *pgxpool.Pool) *ClientRepo {
	return &ClientRepo{db: db}
}

const clientColumns = `id, client_id, client_secret_hash, client_secret_plain, name, description, logo_url, owner_id,
	redirect_uris, grant_types, response_types, scopes, token_endpoint_auth_method,
	min_security_level, protocol_type, is_active, is_public, created_at, updated_at`

func scanClient(row pgx.Row) (*domain.OIDCClient, error) {
	var c domain.OIDCClient
	var ownerID *uuid.UUID
	var isPublic bool
	err := row.Scan(
		&c.ID, &c.ClientID, &c.ClientSecretHash, &c.ClientSecretPlain, &c.ClientName, &c.Description, &c.LogoURL, &ownerID,
		&c.RedirectURIs, &c.GrantTypes, &c.ResponseTypes, &c.Scopes, &c.TokenEndpointAuthMethod,
		&c.MinSecurityLevel, &c.ProtocolType, &c.IsActive, &isPublic, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.OwnerUserID = ownerID
	c.IsConfidential = !isPublic
	return &c, nil
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
			id, client_id, client_secret_hash, client_secret_plain, name, description, logo_url, owner_id,
			redirect_uris, grant_types, response_types, scopes, token_endpoint_auth_method,
			min_security_level, protocol_type, is_active, is_public, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
	`
	_, err := r.db.Exec(ctx, query,
		c.ID, c.ClientID, c.ClientSecretHash, c.ClientSecretPlain, c.ClientName, c.Description, c.LogoURL, c.OwnerUserID,
		c.RedirectURIs, c.GrantTypes, c.ResponseTypes, c.Scopes, c.TokenEndpointAuthMethod,
		c.MinSecurityLevel, c.ProtocolType, c.IsActive, !c.IsConfidential, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert client: %w", err)
	}
	return nil
}

func (r *ClientRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.OIDCClient, error) {
	query := `SELECT ` + clientColumns + ` FROM oidc_clients WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)
	c, err := scanClient(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *ClientRepo) GetByClientID(ctx context.Context, clientID string) (*domain.OIDCClient, error) {
	query := `SELECT ` + clientColumns + ` FROM oidc_clients WHERE client_id = $1`
	row := r.db.QueryRow(ctx, query, clientID)
	c, err := scanClient(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM oidc_clients`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + clientColumns + ` FROM oidc_clients ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
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
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM oidc_clients WHERE owner_id = $1`, ownerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + clientColumns + ` FROM oidc_clients WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, ownerID, limit, offset)
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
			name = $2, description = $3, logo_url = $4,
			redirect_uris = $5, grant_types = $6, response_types = $7, scopes = $8,
			token_endpoint_auth_method = $9, min_security_level = $10, protocol_type = $11,
			is_active = $12, is_public = $13, updated_at = $14
		WHERE id = $1
	`
	tag, err := r.db.Exec(ctx, query,
		c.ID, c.ClientName, c.Description, c.LogoURL,
		c.RedirectURIs, c.GrantTypes, c.ResponseTypes, c.Scopes,
		c.TokenEndpointAuthMethod, c.MinSecurityLevel, c.ProtocolType,
		c.IsActive, !c.IsConfidential, c.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *ClientRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM oidc_clients WHERE id = $1`, id)
	return err
}

func (r *ClientRepo) UpdateSecret(ctx context.Context, id uuid.UUID, hash, plain string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE oidc_clients SET client_secret_hash = $2, client_secret_plain = $3, updated_at = NOW() WHERE id = $1`,
		id, hash, plain,
	)
	return err
}
