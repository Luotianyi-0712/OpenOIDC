package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepo struct {
	db *pgxpool.Pool
}

func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (*domain.GlobalSetting, error) {
	var s domain.GlobalSetting
	var rawValue []byte
	err := r.db.QueryRow(ctx,
		`SELECT key, value, description, updated_at FROM global_settings WHERE key = $1`,
		key,
	).Scan(&s.Key, &rawValue, &s.Description, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	if len(rawValue) > 0 {
		var str string
		if json.Unmarshal(rawValue, &str) == nil {
			s.Value = str
		} else {
			s.Value = string(rawValue)
		}
	}
	return &s, nil
}

func (r *SettingsRepo) Upsert(ctx context.Context, key, value, desc string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO global_settings (key, value, description, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description, updated_at = NOW()`,
		key, raw, desc,
	)
	return err
}

func (r *SettingsRepo) List(ctx context.Context) ([]*domain.GlobalSetting, error) {
	rows, err := r.db.Query(ctx,
		`SELECT key, value, description, updated_at FROM global_settings ORDER BY key`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*domain.GlobalSetting
	for rows.Next() {
		var s domain.GlobalSetting
		var rawValue []byte
		if err := rows.Scan(&s.Key, &rawValue, &s.Description, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if len(rawValue) > 0 {
			var str string
			if json.Unmarshal(rawValue, &str) == nil {
				s.Value = str
			} else {
				s.Value = string(rawValue)
			}
		}
		settings = append(settings, &s)
	}
	return settings, rows.Err()
}
