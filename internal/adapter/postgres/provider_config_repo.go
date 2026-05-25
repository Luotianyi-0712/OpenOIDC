package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProviderConfigRepo struct {
	db *pgxpool.Pool
}

func NewProviderConfigRepo(db *pgxpool.Pool) *ProviderConfigRepo {
	return &ProviderConfigRepo{db: db}
}

const providerConfigColumns = `id, provider, display_name, enabled, client_id, client_secret,
	extra_config, COALESCE(sort_order, 0), created_at, updated_at`

func scanProviderConfig(row pgx.Row) (*domain.ProviderConfig, error) {
	var pc domain.ProviderConfig
	var clientID, clientSecret string
	var extra []byte
	err := row.Scan(
		&pc.ID, &pc.Provider, &pc.DisplayName, &pc.IsEnabled, &clientID, &clientSecret,
		&extra, &pc.SortOrder, &pc.CreatedAt, &pc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if clientID != "" {
		s := clientID
		pc.ClientID = &s
	}
	if clientSecret != "" {
		s := clientSecret
		pc.ClientSecret = &s
	}
	if len(extra) > 0 {
		_ = json.Unmarshal(extra, &pc.ExtraConfig)
	}
	return &pc, nil
}

func (r *ProviderConfigRepo) Get(ctx context.Context, provider string) (*domain.ProviderConfig, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+providerConfigColumns+` FROM provider_configs WHERE provider = $1`,
		provider,
	)
	pc, err := scanProviderConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return pc, nil
}

func (r *ProviderConfigRepo) List(ctx context.Context) ([]*domain.ProviderConfig, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+providerConfigColumns+` FROM provider_configs ORDER BY sort_order, provider`,
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

func (r *ProviderConfigRepo) Upsert(ctx context.Context, pc *domain.ProviderConfig) error {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}
	now := time.Now().UTC()
	if pc.CreatedAt.IsZero() {
		pc.CreatedAt = now
	}
	pc.UpdatedAt = now

	var extra []byte
	if pc.ExtraConfig != nil {
		var err error
		extra, err = json.Marshal(pc.ExtraConfig)
		if err != nil {
			return fmt.Errorf("marshal extra_config: %w", err)
		}
	}

	var clientID, clientSecret string
	if pc.ClientID != nil {
		clientID = *pc.ClientID
	}
	if pc.ClientSecret != nil {
		clientSecret = *pc.ClientSecret
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO provider_configs
		 (id, provider, display_name, enabled, client_id, client_secret, extra_config, sort_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (provider) DO UPDATE SET
		 display_name = EXCLUDED.display_name,
		 enabled = EXCLUDED.enabled,
		 client_id = EXCLUDED.client_id,
		 client_secret = EXCLUDED.client_secret,
		 extra_config = EXCLUDED.extra_config,
		 sort_order = EXCLUDED.sort_order,
		 updated_at = EXCLUDED.updated_at`,
		pc.ID, pc.Provider, pc.DisplayName, pc.IsEnabled, clientID, clientSecret,
		extra, pc.SortOrder, pc.CreatedAt, pc.UpdatedAt,
	)
	return err
}

func (r *ProviderConfigRepo) Delete(ctx context.Context, provider string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM provider_configs WHERE provider = $1`, provider)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}
