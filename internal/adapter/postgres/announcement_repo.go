package postgres

import (
	"context"
	"errors"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnnouncementRepo struct{ db *pgxpool.Pool }

func NewAnnouncementRepo(db *pgxpool.Pool) *AnnouncementRepo { return &AnnouncementRepo{db: db} }

func (r *AnnouncementRepo) List(ctx context.Context, publishedOnly bool) ([]*domain.Announcement, error) {
	query := `SELECT id, title, content, display_mode, dismissible, scrolling, is_published, revision, created_at, updated_at FROM announcements`
	if publishedOnly {
		query += ` WHERE is_published = TRUE`
	}
	rows, err := r.db.Query(ctx, query+` ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.Announcement, 0)
	for rows.Next() {
		var a domain.Announcement
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.DisplayMode, &a.Dismissible, &a.Scrolling, &a.IsPublished, &a.Revision, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, &a)
	}
	return items, rows.Err()
}

func (r *AnnouncementRepo) Create(ctx context.Context, a *domain.Announcement) error {
	_, err := r.db.Exec(ctx, `INSERT INTO announcements
		(id, title, content, display_mode, dismissible, scrolling, is_published, revision, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, a.Title, a.Content, a.DisplayMode, a.Dismissible, a.Scrolling, a.IsPublished, a.Revision, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AnnouncementRepo) Update(ctx context.Context, a *domain.Announcement) error {
	err := r.db.QueryRow(ctx, `UPDATE announcements SET title = $1, content = $2, display_mode = $3, dismissible = $4, scrolling = $5, is_published = $6, revision = $7, updated_at = $8 WHERE id = $9 RETURNING created_at`,
		a.Title, a.Content, a.DisplayMode, a.Dismissible, a.Scrolling, a.IsPublished, a.Revision, a.UpdatedAt, a.ID).Scan(&a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return port.ErrNotFound
	}
	return err
}

func (r *AnnouncementRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

var _ port.AnnouncementRepository = (*AnnouncementRepo)(nil)
