package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type AnnouncementInput struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	DisplayMode string `json:"display_mode"`
	Dismissible bool   `json:"dismissible"`
	Scrolling   bool   `json:"scrolling"`
	IsPublished bool   `json:"is_published"`
}

type AnnouncementService struct {
	repo  port.AnnouncementRepository
	audit port.AuditRepository
}

func NewAnnouncementService(repo port.AnnouncementRepository, audit port.AuditRepository) *AnnouncementService {
	return &AnnouncementService{repo: repo, audit: audit}
}

func (s *AnnouncementService) List(ctx context.Context, publishedOnly bool) ([]*domain.Announcement, error) {
	return s.repo.List(ctx, publishedOnly)
}

func (s *AnnouncementService) Save(ctx context.Context, id, adminID uuid.UUID, input AnnouncementInput) (*domain.Announcement, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" || utf8.RuneCountInString(input.Title) > 160 {
		return nil, fmt.Errorf("%w: title must contain 1 to 160 characters", ErrInvalidInput)
	}
	if input.Content == "" || len(input.Content) > 65536 {
		return nil, fmt.Errorf("%w: content must contain 1 to 65536 bytes", ErrInvalidInput)
	}
	switch input.DisplayMode {
	case "banner", "modal", "both":
	default:
		return nil, fmt.Errorf("%w: display_mode must be banner, modal or both", ErrInvalidInput)
	}
	now := time.Now().UTC()
	a := &domain.Announcement{
		ID: id, Title: input.Title, Content: input.Content, DisplayMode: input.DisplayMode,
		Dismissible: input.Dismissible, Scrolling: input.Scrolling, IsPublished: input.IsPublished,
		Revision: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	action := "announcement.updated"
	if id == uuid.Nil {
		a.ID = uuid.New()
		if err := s.repo.Create(ctx, a); err != nil {
			return nil, err
		}
		action = "announcement.created"
	} else {
		if err := s.repo.Update(ctx, a); err != nil {
			return nil, err
		}
	}
	s.recordAudit(ctx, adminID, a.ID, action)
	return a, nil
}

func (s *AnnouncementService) Delete(ctx context.Context, id, adminID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, adminID, id, "announcement.deleted")
	return nil
}

func (s *AnnouncementService) recordAudit(ctx context.Context, adminID, id uuid.UUID, action string) {
	if s.audit == nil {
		return
	}
	kind, resourceID := "announcement", id.String()
	_ = s.audit.CreateLog(ctx, &domain.AuditLog{
		ID: uuid.New(), UserID: &adminID, Action: action, ResourceType: &kind,
		ResourceID: &resourceID, CreatedAt: time.Now().UTC(),
	})
}
