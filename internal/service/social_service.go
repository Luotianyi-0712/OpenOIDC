package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type SocialService struct {
	bindingRepo  port.BindingRepository
	userRepo     port.UserRepository
	registry     port.SocialProviderRegistry
	cache        port.Cache
	securitySvc  *SecurityLevelService
	sessionRepo  port.SessionRepository
	auditRepo    port.AuditRepository
	settingsRepo port.SettingsRepository
	riskListRepo port.RiskListRepository
	cfg          *config.Config
}

func NewSocialService(
	bindingRepo port.BindingRepository,
	userRepo port.UserRepository,
	registry port.SocialProviderRegistry,
	cache port.Cache,
	securitySvc *SecurityLevelService,
	sessionRepo port.SessionRepository,
	auditRepo port.AuditRepository,
	settingsRepo port.SettingsRepository,
	riskListRepo port.RiskListRepository,
	cfg *config.Config,
) *SocialService {
	return &SocialService{
		bindingRepo:  bindingRepo,
		userRepo:     userRepo,
		registry:     registry,
		cache:        cache,
		securitySvc:  securitySvc,
		sessionRepo:  sessionRepo,
		auditRepo:    auditRepo,
		settingsRepo: settingsRepo,
		riskListRepo: riskListRepo,
		cfg:          cfg,
	}
}

const (
	oauthStateModeBind  = "bind"
	oauthStateModeLogin = "login"
)

type oauthStateData struct {
	Mode     string    `json:"mode"`
	Provider string    `json:"provider"`
	UserID   uuid.UUID `json:"user_id,omitempty"`
	ReturnTo string    `json:"return_to,omitempty"`
}

type OAuthStateInfo struct {
	Mode     string
	Provider string
	UserID   uuid.UUID
	ReturnTo string
}

func (s *SocialService) PeekState(ctx context.Context, state string) (*OAuthStateInfo, error) {
	raw, err := s.cache.GetOAuthState(ctx, state)
	if err != nil {
		return nil, ErrInvalidToken
	}
	var data oauthStateData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, ErrInvalidToken
	}
	return &OAuthStateInfo{
		Mode:     data.Mode,
		Provider: data.Provider,
		UserID:   data.UserID,
		ReturnTo: data.ReturnTo,
	}, nil
}

func (s *SocialService) BeginBinding(ctx context.Context, userID uuid.UUID, provider, returnTo string) (string, error) {
	if !s.registry.IsEnabled(provider) {
		return "", ErrProviderDisabled
	}
	prov, err := s.registry.Get(provider)
	if err != nil {
		return "", ErrProviderDisabled
	}

	existing, err := s.bindingRepo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return "", fmt.Errorf("check binding: %w", err)
	}
	if existing != nil {
		return "", ErrAlreadyBound
	}

	state, err := generateRandomToken(24)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(oauthStateData{
		Mode:     oauthStateModeBind,
		Provider: provider,
		UserID:   userID,
		ReturnTo: returnTo,
	})
	if err := s.cache.SetOAuthState(ctx, state, data, 10*time.Minute); err != nil {
		return "", fmt.Errorf("store state: %w", err)
	}

	redirect := s.callbackURL(provider)
	authURL, err := prov.BeginAuth(ctx, state, redirect)
	if err != nil {
		return "", fmt.Errorf("begin auth: %w", err)
	}
	return authURL, nil
}

func (s *SocialService) CompleteBinding(ctx context.Context, userID uuid.UUID, provider string, r *http.Request) (*domain.SocialBinding, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return nil, ErrInvalidToken
	}
	stateData, err := s.consumeState(ctx, state)
	if err != nil {
		return nil, err
	}
	if stateData.Mode != oauthStateModeBind || stateData.Provider != provider || stateData.UserID != userID {
		return nil, ErrInvalidToken
	}
	prov, err := s.registry.Get(provider)
	if err != nil {
		return nil, ErrProviderDisabled
	}
	info, err := prov.CompleteAuth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("complete auth: %w", err)
	}

	// Check risk list.
	if s.riskListRepo != nil {
		if entry, _ := s.riskListRepo.Check(ctx, provider, info.ProviderUID); entry != nil {
			return nil, fmt.Errorf("%w: this social account is blocked by risk control", ErrPermissionDenied)
		}
	}

	owner, err := s.bindingRepo.GetByProviderUID(ctx, provider, info.ProviderUID)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, fmt.Errorf("check provider uid: %w", err)
	}
	if owner != nil {
		return nil, ErrAlreadyBound
	}

	now := time.Now().UTC()
	var email, name *string
	if info.Email != "" {
		v := info.Email
		email = &v
	}
	if info.DisplayName != "" {
		v := info.DisplayName
		name = &v
	}
	binding := &domain.SocialBinding{
		ID:            uuid.New(),
		UserID:        userID,
		Provider:      provider,
		ProviderUID:   info.ProviderUID,
		ProviderEmail: email,
		ProviderName:  name,
		RawProfile:    info.RawProfile,
		BoundAt:       now,
		VerifiedAt:    &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.bindingRepo.Create(ctx, binding); err != nil {
		return nil, fmt.Errorf("create binding: %w", err)
	}

	if s.securitySvc != nil {
		if _, err := s.securitySvc.ComputeSecurityLevel(ctx, userID); err != nil {
			return nil, fmt.Errorf("recompute security level: %w", err)
		}
	}

	rt := "social_binding"
	rid := binding.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       "social.bound",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"provider": provider, "provider_uid": info.ProviderUID},
		CreatedAt:    now,
	})

	return binding, nil
}

func (s *SocialService) BeginSocialLogin(ctx context.Context, provider, returnTo string) (string, error) {
	if !s.registry.IsEnabled(provider) {
		return "", ErrProviderDisabled
	}
	prov, err := s.registry.Get(provider)
	if err != nil {
		return "", ErrProviderDisabled
	}
	state, err := generateRandomToken(24)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(oauthStateData{
		Mode:     oauthStateModeLogin,
		Provider: provider,
		ReturnTo: returnTo,
	})
	if err := s.cache.SetOAuthState(ctx, state, data, 10*time.Minute); err != nil {
		return "", fmt.Errorf("store state: %w", err)
	}
	redirect := s.callbackURL(provider)
	return prov.BeginAuth(ctx, state, redirect)
}

func (s *SocialService) CompleteSocialLogin(ctx context.Context, provider string, r *http.Request, ip, userAgent string) (*domain.UserSession, *domain.User, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return nil, nil, ErrInvalidToken
	}
	stateData, err := s.consumeState(ctx, state)
	if err != nil {
		return nil, nil, err
	}
	if stateData.Mode != oauthStateModeLogin || stateData.Provider != provider {
		return nil, nil, ErrInvalidToken
	}
	prov, err := s.registry.Get(provider)
	if err != nil {
		return nil, nil, ErrProviderDisabled
	}
	info, err := prov.CompleteAuth(ctx, r)
	if err != nil {
		return nil, nil, fmt.Errorf("complete auth: %w", err)
	}

	// Check risk list.
	if s.riskListRepo != nil {
		if entry, _ := s.riskListRepo.Check(ctx, provider, info.ProviderUID); entry != nil {
			return nil, nil, fmt.Errorf("%w: this social account is blocked by risk control", ErrPermissionDenied)
		}
	}

	now := time.Now().UTC()
	var user *domain.User

	binding, err := s.bindingRepo.GetByProviderUID(ctx, provider, info.ProviderUID)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, nil, fmt.Errorf("lookup binding: %w", err)
	}

	if binding != nil {
		user, err = s.userRepo.GetByID(ctx, binding.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("lookup user: %w", err)
		}
	} else {
		if !s.isSettingEnabled(ctx, "registration_enabled") {
			return nil, nil, ErrRegistrationDisabled
		}

		if info.Email != "" {
			// Check email domain whitelist for social registration.
			if err := s.checkEmailDomainAllowed(ctx, info.Email); err != nil {
				return nil, nil, err
			}

			user, err = s.userRepo.GetByEmail(ctx, strings.ToLower(info.Email))
			if err != nil && !errors.Is(err, port.ErrNotFound) {
				return nil, nil, fmt.Errorf("lookup user by email: %w", err)
			}
		}

		if user == nil {
			if info.Email == "" {
				return nil, nil, ErrBindingNotFound
			}
			displayName := info.DisplayName
			if displayName == "" {
				displayName = strings.SplitN(info.Email, "@", 2)[0]
			}
			user = &domain.User{
				ID:            uuid.New(),
				Email:         strings.ToLower(info.Email),
				EmailVerified: info.EmailVerified,
				DisplayName:   displayName,
				AvatarURL:     info.AvatarURL,
				Status:        domain.UserStatusActive,
				SecurityLevel: 0,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := s.userRepo.Create(ctx, user); err != nil {
				return nil, nil, fmt.Errorf("create user: %w", err)
			}
			s.audit(ctx, &user.ID, "user.register_social", "user", user.ID.String(), &ip, map[string]any{
				"provider": provider,
				"email":    user.Email,
			})
		}

		var provEmail, provName *string
		if info.Email != "" {
			v := info.Email
			provEmail = &v
		}
		if info.DisplayName != "" {
			v := info.DisplayName
			provName = &v
		}
		newBinding := &domain.SocialBinding{
			ID:            uuid.New(),
			UserID:        user.ID,
			Provider:      provider,
			ProviderUID:   info.ProviderUID,
			ProviderEmail: provEmail,
			ProviderName:  provName,
			RawProfile:    info.RawProfile,
			BoundAt:       now,
			VerifiedAt:    &now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.bindingRepo.Create(ctx, newBinding); err != nil {
			return nil, nil, fmt.Errorf("create binding: %w", err)
		}

		if s.securitySvc != nil {
			_, _ = s.securitySvc.ComputeSecurityLevel(ctx, user.ID)
		}
	}

	switch user.Status {
	case domain.UserStatusSuspended:
		return nil, nil, ErrAccountSuspended
	case domain.UserStatusDeleted:
		return nil, nil, ErrAccountDeleted
	}

	session, err := s.createSession(ctx, user.ID, ip, userAgent)
	if err != nil {
		return nil, nil, err
	}
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, nil, fmt.Errorf("update last login: %w", err)
	}

	s.audit(ctx, &user.ID, "user.login_social", "user", user.ID.String(), &ip, map[string]any{
		"provider": provider,
	})

	return session, user, nil
}

func (s *SocialService) createSession(ctx context.Context, userID uuid.UUID, ip, userAgent string) (*domain.UserSession, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}
	ttl := s.cfg.Session.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	now := time.Now().UTC()
	var ipPtr, uaPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	if userAgent != "" {
		uaPtr = &userAgent
	}
	session := &domain.UserSession{
		ID:           uuid.New(),
		UserID:       userID,
		SessionToken: token,
		IPAddress:    ipPtr,
		UserAgent:    uaPtr,
		ExpiresAt:    now.Add(ttl),
		CreatedAt:    now,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

func (s *SocialService) isSettingEnabled(ctx context.Context, key string) bool {
	setting, err := s.settingsRepo.Get(ctx, key)
	if err != nil {
		return true
	}
	return setting.Value != "false"
}

// checkEmailDomainAllowed validates that the email domain is in the allowed list.
// If the setting is empty/unset, all domains are allowed.
func (s *SocialService) checkEmailDomainAllowed(ctx context.Context, email string) error {
	if s.settingsRepo == nil {
		return nil
	}
	setting, err := s.settingsRepo.Get(ctx, "allowed_email_domains")
	if err != nil || setting.Value == "" {
		return nil // No restriction.
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}
	domain := strings.ToLower(parts[1])
	for _, d := range strings.Split(setting.Value, ",") {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" && d == domain {
			return nil
		}
	}
	return fmt.Errorf("%w: email domain not allowed", ErrInvalidInput)
}

func (s *SocialService) audit(ctx context.Context, userID *uuid.UUID, action, resourceType, resourceID string, ip *string, details map[string]any) {
	rt := resourceType
	rid := resourceID
	log := &domain.AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		IPAddress: ip,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	}
	if rt != "" {
		log.ResourceType = &rt
	}
	if rid != "" {
		log.ResourceID = &rid
	}
	_ = s.auditRepo.CreateLog(ctx, log)
}

func (s *SocialService) Unbind(ctx context.Context, userID uuid.UUID, provider string) error {
	existing, err := s.bindingRepo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrBindingNotFound
		}
		return fmt.Errorf("lookup binding: %w", err)
	}
	if err := s.bindingRepo.Delete(ctx, userID, provider); err != nil {
		return fmt.Errorf("delete binding: %w", err)
	}
	if s.securitySvc != nil {
		if _, err := s.securitySvc.ComputeSecurityLevel(ctx, userID); err != nil {
			return fmt.Errorf("recompute security level: %w", err)
		}
	}
	rt := "social_binding"
	rid := existing.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       "social.unbound",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"provider": provider},
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *SocialService) ListBindings(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error) {
	return s.bindingRepo.ListByUser(ctx, userID)
}

func (s *SocialService) consumeState(ctx context.Context, state string) (*oauthStateData, error) {
	raw, err := s.cache.GetOAuthState(ctx, state)
	if err != nil {
		return nil, ErrInvalidToken
	}
	_ = s.cache.DeleteOAuthState(ctx, state)
	var data oauthStateData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, ErrInvalidToken
	}
	return &data, nil
}

func (s *SocialService) callbackURL(provider string) string {
	base := s.cfg.Server.BaseURL
	if base == "" {
		base = s.cfg.Server.Issuer
	}
	return fmt.Sprintf("%s/api/v1/social/%s/callback", base, provider)
}
