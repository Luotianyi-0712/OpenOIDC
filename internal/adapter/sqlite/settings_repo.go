package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

// SettingsRepo implements port.SettingsRepository using SQLite.
type SettingsRepo struct {
	db *sql.DB
}

// NewSettingsRepo returns a new SettingsRepo.
func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// Get retrieves a global setting by key.
func (r *SettingsRepo) Get(ctx context.Context, key string) (*domain.GlobalSetting, error) {
	var s domain.GlobalSetting
	var updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT key, value, description, updated_at FROM global_settings WHERE key = ?`,
		key,
	).Scan(&s.Key, &s.Value, &s.Description, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	s.UpdatedAt = parseTimeLoose(updatedAt)
	return &s, nil
}

// Upsert inserts or replaces a global setting.
func (r *SettingsRepo) Upsert(ctx context.Context, key, value, desc string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO global_settings (key, value, description, updated_at)
		 VALUES (?, ?, ?, ?)`,
		key, value, desc, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// List returns all global settings ordered by key.
func (r *SettingsRepo) List(ctx context.Context) ([]*domain.GlobalSetting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, value, description, updated_at FROM global_settings ORDER BY key`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*domain.GlobalSetting
	for rows.Next() {
		var s domain.GlobalSetting
		var updatedAt string
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &updatedAt); err != nil {
			return nil, err
		}
		s.UpdatedAt = parseTimeLoose(updatedAt)
		settings = append(settings, &s)
	}
	return settings, rows.Err()
}

// parseTimeLoose tries RFC3339 first, then Go's default time.String() format.
func parseTimeLoose(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05.9999999 -0700 MST", s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	return time.Time{}
}
