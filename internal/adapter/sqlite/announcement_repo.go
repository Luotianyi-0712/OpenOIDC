package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type AnnouncementRepo struct{ db *sql.DB }

func NewAnnouncementRepo(db *sql.DB) *AnnouncementRepo { return &AnnouncementRepo{db: db} }

func (r *AnnouncementRepo) List(ctx context.Context, publishedOnly bool) ([]*domain.Announcement, error) {
	query := `SELECT id, title, content, display_mode, dismissible, scrolling, is_published, revision, created_at, updated_at FROM announcements`
	if publishedOnly {
		query += ` WHERE is_published = 1`
	}
	rows, err := r.db.QueryContext(ctx, query+` ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.Announcement, 0)
	for rows.Next() {
		var a domain.Announcement
		var created, updated string
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.DisplayMode, &a.Dismissible, &a.Scrolling, &a.IsPublished, &a.Revision, &created, &updated); err != nil {
			return nil, err
		}
		a.CreatedAt = parseTimeLoose(created)
		a.UpdatedAt = parseTimeLoose(updated)
		items = append(items, &a)
	}
	return items, rows.Err()
}

func (r *AnnouncementRepo) Create(ctx context.Context, a *domain.Announcement) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO announcements
		(id, title, content, display_mode, dismissible, scrolling, is_published, revision, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID.String(), a.Title, a.Content, a.DisplayMode, a.Dismissible, a.Scrolling, a.IsPublished, a.Revision.String(), a.CreatedAt.Format(time.RFC3339Nano), a.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *AnnouncementRepo) Update(ctx context.Context, a *domain.Announcement) error {
	var created string
	err := r.db.QueryRowContext(ctx, `UPDATE announcements SET title = ?, content = ?, display_mode = ?, dismissible = ?, scrolling = ?, is_published = ?, revision = ?, updated_at = ? WHERE id = ? RETURNING created_at`,
		a.Title, a.Content, a.DisplayMode, a.Dismissible, a.Scrolling, a.IsPublished, a.Revision.String(), a.UpdatedAt.Format(time.RFC3339Nano), a.ID.String()).Scan(&created)
	if errors.Is(err, sql.ErrNoRows) {
		return port.ErrNotFound
	}
	if err != nil {
		return err
	}
	a.CreatedAt = parseTimeLoose(created)
	return nil
}

func (r *AnnouncementRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM announcements WHERE id = ?`, id.String())
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count == 0 {
		return port.ErrNotFound
	}
	return nil
}

var _ port.AnnouncementRepository = (*AnnouncementRepo)(nil)
