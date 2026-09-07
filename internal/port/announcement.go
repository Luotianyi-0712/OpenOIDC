package port

import (
	"context"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
)

type AnnouncementRepository interface {
	List(ctx context.Context, publishedOnly bool) ([]*domain.Announcement, error)
	Create(ctx context.Context, announcement *domain.Announcement) error
	Update(ctx context.Context, announcement *domain.Announcement) error
	Delete(ctx context.Context, id uuid.UUID) error
}
