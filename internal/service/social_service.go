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
	Intent   string    `json:"intent,omitempty"`
	UserID   uuid.UUID `json:"user_id,omitempty"`
	ReturnTo string    `json:"return_to,omitempty"`
}

type OAuthStateInfo struct {
	Mode     string
	Provider string
	Intent   string
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
		Intent:   data.Intent,
		UserID:   data.UserID,
		ReturnTo: data.ReturnTo,
	}, nil
}

func (s *SocialService) BeginBinding(ctx context.Context, userID uuid.UUID, provider, returnTo string) (string, error) {
	if !s.isSettingEnabled(ctx, "social_login_enabled") {
		return "", ErrSocialLoginDisabled
	}
	if !s.isSettingEnabled(ctx, "social_binding_enabled") {
		return "", ErrSocialBindingDisabled
	}
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
	if !s.isSettingEnabled(ctx, "social_login_enabled") {
		return nil, ErrSocialLoginDisabled
	}
	if !s.isSettingEnabled(ctx, "social_binding_enabled") {
		return nil, ErrSocialBindingDisabled
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
	binding := s.bindingFromProviderInfo(userID, provider, info, now)
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
		Details:      map[string]any{"provider": provider, "provider_uid": info.ProviderUID, "auth_status": binding.LastAuthStatus},
		CreatedAt:    now,
	})

	return binding, nil
}

func (s *SocialService) BeginSocialLogin(ctx context.Context, provider, returnTo, intent string) (string, error) {
	if !s.isSettingEnabled(ctx, "social_login_enabled") {
		return "", ErrSocialLoginDisabled
	}
	if !s.registry.IsEnabled(provider) {
		return "", ErrProviderDisabled
	}
	if intent == "register" {
		if !s.registry.IsRegisterEnabled(provider) {
			return "", ErrSocialRegistrationDisabled
		}
	} else if !s.registry.IsLoginEnabled(provider) {
		return "", ErrSocialLoginDisabled
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
		Intent:   intent,
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
	if !s.isSettingEnabled(ctx, "social_login_enabled") {
		return nil, nil, ErrSocialLoginDisabled
	}
	if stateData.Intent == "register" {
		if !s.registry.IsRegisterEnabled(provider) {
			return nil, nil, ErrSocialRegistrationDisabled
		}
	} else if !s.registry.IsLoginEnabled(provider) {
		return nil, nil, ErrSocialLoginDisabled
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
		if err := s.updateBindingFromProviderInfo(ctx, binding, info, now); err != nil {
			return nil, nil, fmt.Errorf("update binding auth status: %w", err)
		}
	} else {
		if !s.isSettingEnabled(ctx, "registration_enabled") {
			return nil, nil, ErrRegistrationDisabled
		}
		if !s.isSettingEnabled(ctx, "social_register_enabled") {
			return nil, nil, ErrSocialRegistrationDisabled
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
				Role:          domain.RoleUser,
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

		newBinding := s.bindingFromProviderInfo(user.ID, provider, info, now)
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
	if err := s.bindingRepo.SoftUnbind(ctx, userID, provider, "user_request"); err != nil {
		return fmt.Errorf("soft unbind: %w", err)
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
		Details:      map[string]any{"provider": provider, "provider_uid": existing.ProviderUID, "reason": "user_request"},
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *SocialService) ListBindings(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error) {
	return s.bindingRepo.ListByUser(ctx, userID)
}

func (s *SocialService) SyncAuthorizationStatus(ctx context.Context, limit int, staleBefore time.Time) error {
	bindings, err := s.bindingRepo.ListDueAuthChecks(ctx, staleBefore, limit)
	if err != nil {
		return fmt.Errorf("list due auth checks: %w", err)
	}
	for _, binding := range bindings {
		if err := s.checkAuthorization(ctx, binding); err != nil {
			return err
		}
	}
	return nil
}

func (s *SocialService) checkAuthorization(ctx context.Context, binding *domain.SocialBinding) error {
	prov, err := s.registry.Get(binding.Provider)
	now := time.Now().UTC()
	if err != nil || !s.registry.IsEnabled(binding.Provider) {
		binding.LastAuthCheckAt = &now
		binding.LastAuthStatus = domain.SocialAuthStatusUnknown
		msg := "provider disabled or unavailable"
		binding.LastAuthError = &msg
		return s.bindingRepo.Update(ctx, binding)
	}
	if !prov.SupportsRefresh() && binding.AccessToken == nil {
		binding.LastAuthCheckAt = &now
		binding.LastAuthStatus = domain.SocialAuthStatusUnsupported
		binding.LastAuthError = nil
		return s.bindingRepo.Update(ctx, binding)
	}

	if binding.RefreshToken != nil && *binding.RefreshToken != "" && prov.SupportsRefresh() {
		token, refreshErr := prov.RefreshToken(ctx, *binding.RefreshToken)
		if refreshErr != nil {
			if isAuthorizationLostError(refreshErr) {
				return s.markAuthorizationLost(ctx, binding, domain.SocialBindingStatusProviderRevoked, domain.SocialAuthStatusRevoked, refreshErr)
			}
			return s.markAuthorizationUnknown(ctx, binding, refreshErr)
		}
		s.applyToken(binding, token)
		binding.LastAuthCheckAt = &now
		binding.LastAuthStatus = domain.SocialAuthStatusActive
		binding.LastAuthError = nil
		return s.bindingRepo.Update(ctx, binding)
	}

	if binding.AccessToken == nil || *binding.AccessToken == "" {
		binding.LastAuthCheckAt = &now
		binding.LastAuthStatus = domain.SocialAuthStatusUnsupported
		binding.LastAuthError = nil
		return s.bindingRepo.Update(ctx, binding)
	}
	validator, ok := prov.(port.TokenValidatingProvider)
	if !ok {
		binding.LastAuthCheckAt = &now
		binding.LastAuthStatus = domain.SocialAuthStatusUnsupported
		binding.LastAuthError = nil
		return s.bindingRepo.Update(ctx, binding)
	}
	info, validateErr := validator.ValidateToken(ctx, *binding.AccessToken)
	if validateErr != nil {
		if isAuthorizationLostError(validateErr) || (binding.TokenExpiry != nil && binding.TokenExpiry.Before(now)) {
			status := domain.SocialBindingStatusProviderRevoked
			authStatus := domain.SocialAuthStatusRevoked
			if binding.TokenExpiry != nil && binding.TokenExpiry.Before(now) {
				status = domain.SocialBindingStatusTokenExpired
				authStatus = domain.SocialAuthStatusExpired
			}
			return s.markAuthorizationLost(ctx, binding, status, authStatus, validateErr)
		}
		return s.markAuthorizationUnknown(ctx, binding, validateErr)
	}
	s.applyProviderSnapshot(binding, info)
	binding.LastAuthCheckAt = &now
	binding.LastAuthStatus = domain.SocialAuthStatusActive
	binding.LastAuthError = nil
	return s.bindingRepo.Update(ctx, binding)
}

func (s *SocialService) markAuthorizationUnknown(ctx context.Context, binding *domain.SocialBinding, cause error) error {
	now := time.Now().UTC()
	binding.LastAuthCheckAt = &now
	binding.LastAuthStatus = domain.SocialAuthStatusUnknown
	msg := cause.Error()
	binding.LastAuthError = &msg
	return s.bindingRepo.Update(ctx, binding)
}

func isAuthorizationLostError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	markers := []string{
		"invalid_grant",
		"invalid token",
		"token expired",
		"expired token",
		"revoked",
		"unauthorized",
		"forbidden",
		"http 401",
		"http 403",
	}
	for _, marker := range markers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func (s *SocialService) markAuthorizationLost(ctx context.Context, binding *domain.SocialBinding, status domain.SocialBindingStatus, authStatus domain.SocialAuthStatus, cause error) error {
	now := time.Now().UTC()
	binding.Status = status
	binding.UnboundAt = &now
	reason := string(authStatus)
	binding.UnbindReason = &reason
	binding.LastAuthCheckAt = &now
	binding.LastAuthStatus = authStatus
	msg := cause.Error()
	binding.LastAuthError = &msg
	binding.AccessToken = nil
	binding.RefreshToken = nil
	binding.TokenExpiry = nil
	binding.TokenType = nil
	binding.TokenScopes = nil
	if err := s.bindingRepo.Update(ctx, binding); err != nil {
		return err
	}
	if s.securitySvc != nil {
		_, _ = s.securitySvc.ComputeSecurityLevel(ctx, binding.UserID)
	}
	s.audit(ctx, &binding.UserID, "social.authorization_lost", "social_binding", binding.ID.String(), nil, map[string]any{
		"provider":      binding.Provider,
		"provider_uid":  binding.ProviderUID,
		"status":        status,
		"auth_status":   authStatus,
		"last_auth_err": msg,
	})
	return nil
}

func (s *SocialService) bindingFromProviderInfo(userID uuid.UUID, provider string, info *port.ProviderUserInfo, now time.Time) *domain.SocialBinding {
	binding := &domain.SocialBinding{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       provider,
		ProviderUID:    info.ProviderUID,
		RawProfile:     info.RawProfile,
		BoundAt:        now,
		VerifiedAt:     &now,
		Status:         domain.SocialBindingStatusActive,
		LastAuthStatus: domain.SocialAuthStatusUnsupported,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.applyProviderSnapshot(binding, info)
	s.applyToken(binding, info.Token)
	return binding
}

func (s *SocialService) updateBindingFromProviderInfo(ctx context.Context, binding *domain.SocialBinding, info *port.ProviderUserInfo, now time.Time) error {
	binding.Status = domain.SocialBindingStatusActive
	binding.UnboundAt = nil
	binding.UnbindReason = nil
	binding.VerifiedAt = &now
	s.applyProviderSnapshot(binding, info)
	s.applyToken(binding, info.Token)
	return s.bindingRepo.Update(ctx, binding)
}

func (s *SocialService) applyProviderSnapshot(binding *domain.SocialBinding, info *port.ProviderUserInfo) {
	if info == nil {
		return
	}
	if info.Email != "" {
		binding.ProviderEmail = ptrString(info.Email)
	}
	if info.DisplayName != "" {
		binding.ProviderName = ptrString(info.DisplayName)
	}
	if info.AvatarURL != "" {
		binding.ProviderAvatar = ptrString(info.AvatarURL)
	}
	if info.RawProfile != nil {
		binding.RawProfile = info.RawProfile
	}
}

func (s *SocialService) applyToken(binding *domain.SocialBinding, token *port.ProviderTokenInfo) {
	if token == nil {
		if binding.LastAuthStatus == "" {
			binding.LastAuthStatus = domain.SocialAuthStatusUnsupported
		}
		return
	}
	if token.AccessToken != "" {
		binding.AccessToken = ptrString(token.AccessToken)
	}
	if token.RefreshToken != "" {
		binding.RefreshToken = ptrString(token.RefreshToken)
	}
	binding.TokenExpiry = token.Expiry
	if token.TokenType != "" {
		binding.TokenType = ptrString(token.TokenType)
	}
	binding.TokenScopes = append([]string(nil), token.Scopes...)
	binding.LastAuthStatus = domain.SocialAuthStatusActive
	binding.LastAuthError = nil
}

func ptrString(v string) *string {
	return &v
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
