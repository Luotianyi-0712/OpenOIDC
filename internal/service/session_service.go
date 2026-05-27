package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type SessionService struct {
	sessionRepo port.SessionRepository
	userRepo    port.UserRepository
	cfg         *config.Config
}

func NewSessionService(sessionRepo port.SessionRepository, userRepo port.UserRepository, cfg *config.Config) *SessionService {
	return &SessionService{sessionRepo: sessionRepo, userRepo: userRepo, cfg: cfg}
}

func (s *SessionService) ValidateSession(ctx context.Context, token string) (*domain.UserSession, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}
	session, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("lookup session: %w", err)
	}
	if session.ExpiresAt.Before(time.Now().UTC()) {
		_ = s.sessionRepo.Delete(ctx, session.ID)
		return nil, ErrSessionExpired
	}
	if s.userRepo != nil {
		user, err := s.userRepo.GetByID(ctx, session.UserID)
		if err != nil {
			_ = s.sessionRepo.Delete(ctx, session.ID)
			return nil, ErrSessionNotFound
		}
		if user.Status != domain.UserStatusActive {
			_ = s.sessionRepo.Delete(ctx, session.ID)
			return nil, ErrAccessDenied
		}
	}
	return session, nil
}

func (s *SessionService) ListSessions(ctx context.Context, userID uuid.UUID) ([]*domain.UserSession, error) {
	return s.sessionRepo.ListByUser(ctx, userID)
}

func (s *SessionService) RevokeSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	sessions, err := s.sessionRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	for _, sess := range sessions {
		if sess.ID == sessionID {
			return s.sessionRepo.Delete(ctx, sessionID)
		}
	}
	return ErrSessionNotFound
}

func (s *SessionService) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return s.sessionRepo.DeleteByUser(ctx, userID)
}

func (s *SessionService) RevokeByToken(ctx context.Context, token string) error {
	sess, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil
	}
	return s.sessionRepo.Delete(ctx, sess.ID)
}

func (s *SessionService) CleanupExpired(ctx context.Context) error {
	return s.sessionRepo.DeleteExpired(ctx)
}
