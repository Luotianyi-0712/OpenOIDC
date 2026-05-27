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

// ProviderConfigRepo implements port.ProviderConfigRepository using SQLite.
type ProviderConfigRepo struct {
	db *sql.DB
}

// NewProviderConfigRepo returns a new ProviderConfigRepo.
func NewProviderConfigRepo(db *sql.DB) *ProviderConfigRepo {
	return &ProviderConfigRepo{db: db}
}

const providerConfigColumns = `id, provider, display_name, is_enabled, client_id, client_secret,
	scopes, redirect_path, extra_config, sort_order, created_at, updated_at`

func scanProviderConfig(row interface{ Scan(...any) error }) (*domain.ProviderConfig, error) {
	var pc domain.ProviderConfig
	var id string
	var clientID, clientSecret sql.NullString
	var scopes string
	var extra sql.NullString

	err := row.Scan(
		&id, &pc.Provider, &pc.DisplayName, &pc.IsEnabled,
		&clientID, &clientSecret, &scopes, &pc.RedirectPath, &extra, &pc.SortOrder,
		&pc.CreatedAt, &pc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	pc.ID = uuid.MustParse(id)
	pc.ClientID = fromNullString(clientID)
	pc.ClientSecret = fromNullString(clientSecret)
	_ = json.Unmarshal([]byte(scopes), &pc.Scopes)
	if pc.Scopes == nil {
		pc.Scopes = []string{}
	}

	if extra.Valid && extra.String != "" {
		_ = json.Unmarshal([]byte(extra.String), &pc.ExtraConfig)
	}
	return &pc, nil
}

// Get retrieves a provider config by provider name.
func (r *ProviderConfigRepo) Get(ctx context.Context, provider string) (*domain.ProviderConfig, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+providerConfigColumns+` FROM provider_configs WHERE provider = ?`,
		provider,
	)
	pc, err := scanProviderConfig(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return pc, nil
}

// List returns all provider configs ordered by sort_order.
func (r *ProviderConfigRepo) List(ctx context.Context) ([]*domain.ProviderConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+providerConfigColumns+` FROM provider_configs ORDER BY sort_order`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*domain.ProviderConfig
	for rows.Next() {
		pc, err := scanProviderConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, pc)
	}
	return configs, rows.Err()
}

// Upsert inserts or replaces a provider config.
func (r *ProviderConfigRepo) Upsert(ctx context.Context, pc *domain.ProviderConfig) error {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}
	now := time.Now().UTC()
	if pc.CreatedAt.IsZero() {
		pc.CreatedAt = now
	}
	if pc.Scopes == nil {
		pc.Scopes = []string{}
	}
	pc.UpdatedAt = now

	var extraStr sql.NullString
	if pc.ExtraConfig != nil {
		b, err := json.Marshal(pc.ExtraConfig)
		if err != nil {
			return fmt.Errorf("marshal extra_config: %w", err)
		}
		extraStr = sql.NullString{String: string(b), Valid: true}
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO provider_configs
		 (id, provider, display_name, is_enabled, client_id, client_secret, scopes, redirect_path, extra_config, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		pc.ID.String(), pc.Provider, pc.DisplayName, pc.IsEnabled,
		toNullString(pc.ClientID), toNullString(pc.ClientSecret),
		marshalStringSlice(pc.Scopes), pc.RedirectPath, extraStr, pc.SortOrder, pc.CreatedAt, pc.UpdatedAt,
	)
	return err
}

func (r *ProviderConfigRepo) Delete(ctx context.Context, provider string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM provider_configs WHERE provider = ?`, provider)
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
