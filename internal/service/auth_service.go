package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type AuthService struct {
	userRepo     port.UserRepository
	sessionRepo  port.SessionRepository
	cache        port.Cache
	auditRepo    port.AuditRepository
	emailSender  port.EmailSender
	settingsRepo port.SettingsRepository
	cfg          *config.Config
}

func NewAuthService(
	userRepo port.UserRepository,
	sessionRepo port.SessionRepository,
	cache port.Cache,
	auditRepo port.AuditRepository,
	emailSender port.EmailSender,
	settingsRepo port.SettingsRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		cache:        cache,
		auditRepo:    auditRepo,
		emailSender:  emailSender,
		settingsRepo: settingsRepo,
		cfg:          cfg,
	}
}

func (s *AuthService) SendRegisterCode(ctx context.Context, email string) error {
	if !s.isSettingEnabled(ctx, "registration_enabled") {
		return ErrRegistrationDisabled
	}
	if !s.isSettingEnabled(ctx, "registration_email_verification_required") {
		return nil
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	if err := s.validateEmailDomain(ctx, email); err != nil {
		return err
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return fmt.Errorf("lookup user: %w", err)
	}
	if existing != nil {
		return ErrAlreadyExists
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}
	if err := s.cache.Set(ctx, registerCodeKey(email), []byte(code), 10*time.Minute); err != nil {
		return fmt.Errorf("store register code: %w", err)
	}
	if s.emailSender != nil {
		return s.emailSender.SendRegistrationCode(ctx, email, code)
	}
	return nil
}

func (s *AuthService) Register(ctx context.Context, email, password, displayName, code string) (*domain.User, error) {
	if !s.isSettingEnabled(ctx, "registration_enabled") {
		return nil, ErrRegistrationDisabled
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if err := s.validatePassword(ctx, password); err != nil {
		return nil, err
	}
	if err := s.validateEmailDomain(ctx, email); err != nil {
		return nil, err
	}

	requireRegisterCode := s.isSettingEnabled(ctx, "registration_email_verification_required")
	if requireRegisterCode {
		cachedCode, err := s.cache.Get(ctx, registerCodeKey(email))
		if err != nil || strings.TrimSpace(code) == "" || string(cachedCode) != strings.TrimSpace(code) {
			return nil, ErrInvalidToken
		}
	}

	if displayName == "" {
		displayName = strings.SplitN(email, "@", 2)[0]
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if existing != nil {
		return nil, ErrAlreadyExists
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:            uuid.New(),
		Email:         email,
		EmailVerified: true,
		PasswordHash:  hash,
		DisplayName:   displayName,
		Status:        domain.UserStatusActive,
		SecurityLevel: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	_ = s.cache.Delete(ctx, registerCodeKey(email))

	s.audit(ctx, &user.ID, "user.register", "user", user.ID.String(), nil, map[string]any{
		"email": email,
	})
	s.audit(ctx, &user.ID, "user.email_verified", "user", user.ID.String(), nil, nil)

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (*domain.UserSession, error) {
	if !s.isSettingEnabled(ctx, "password_login_enabled") {
		return nil, ErrPasswordLoginDisabled
	}

	email = strings.ToLower(strings.TrimSpace(email))

	// Check login lockout before anything else.
	if s.isLockedOut(ctx, email) {
		return nil, ErrAccountLockedOut
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	ok, err := verifyPassword(user.PasswordHash, password)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		s.recordFailedAttempt(ctx, email)
		s.audit(ctx, &user.ID, "user.login_failed", "user", user.ID.String(), &ip, map[string]any{
			"reason": "invalid_password",
		})
		return nil, ErrInvalidCredentials
	}

	switch user.Status {
	case domain.UserStatusSuspended:
		return nil, ErrAccountSuspended
	case domain.UserStatusDeleted:
		return nil, ErrAccountDeleted
	}

	// Clear failed attempts on successful login.
	s.clearFailedAttempts(ctx, email)

	session, err := s.createSession(ctx, user.ID, ip, userAgent)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("update last login: %w", err)
	}

	s.audit(ctx, &user.ID, "user.login", "user", user.ID.String(), &ip, map[string]any{
		"user_agent": userAgent,
	})

	return session, nil
}

// isLockedOut checks if too many failed login attempts have occurred.
func (s *AuthService) isLockedOut(ctx context.Context, email string) bool {
	maxAttempts := s.cfg.Security.MaxLoginAttempts
	if maxAttempts <= 0 {
		return false // Lockout disabled.
	}
	key := "login_attempts:" + email
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return false
	}
	count := int(data[0])
	if len(data) >= 4 {
		count = int(data[0]) | int(data[1])<<8 | int(data[2])<<16 | int(data[3])<<24
	}
	return count >= maxAttempts
}

// recordFailedAttempt increments the failed login counter.
func (s *AuthService) recordFailedAttempt(ctx context.Context, email string) {
	maxAttempts := s.cfg.Security.MaxLoginAttempts
	if maxAttempts <= 0 {
		return
	}
	lockoutDuration := s.cfg.Security.LockoutDuration
	if lockoutDuration <= 0 {
		lockoutDuration = 15 * time.Minute
	}
	key := "login_attempts:" + email
	_, _ = s.cache.IncrementRateLimit(ctx, key, lockoutDuration)
}

// clearFailedAttempts removes the failed login counter after a successful login.
func (s *AuthService) clearFailedAttempts(ctx context.Context, email string) {
	maxAttempts := s.cfg.Security.MaxLoginAttempts
	if maxAttempts <= 0 {
		return
	}
	key := "login_attempts:" + email
	_ = s.cache.Delete(ctx, key)
}

func (s *AuthService) Logout(ctx context.Context, sessionToken string) error {
	session, err := s.sessionRepo.GetByToken(ctx, sessionToken)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("lookup session: %w", err)
	}
	if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	s.audit(ctx, &session.UserID, "user.logout", "session", session.ID.String(), nil, nil)
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	userID, err := s.cache.GetEmailVerifyToken(ctx, token)
	if err != nil {
		return ErrInvalidToken
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	if user.EmailVerified {
		return nil
	}
	user.EmailVerified = true
	user.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	s.audit(ctx, &user.ID, "user.email_verified", "user", user.ID.String(), nil, nil)
	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("lookup user: %w", err)
	}
	token, err := generateRandomToken(32)
	if err != nil {
		return err
	}
	if err := s.cache.SetPasswordResetToken(ctx, token, user.ID, 1*time.Hour); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	if s.emailSender != nil {
		if err := s.emailSender.SendPasswordResetEmail(ctx, user.Email, token); err != nil {
			// Log but don't fail the request (to not reveal user existence).
			_ = err
		}
	}

	s.audit(ctx, &user.ID, "user.password_reset_requested", "user", user.ID.String(), nil, nil)
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	userID, err := s.cache.GetPasswordResetToken(ctx, token)
	if err != nil {
		return ErrInvalidToken
	}
	if err := s.validatePassword(ctx, newPassword); err != nil {
		return err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.sessionRepo.DeleteByUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	s.audit(ctx, &userID, "user.password_reset", "user", userID.String(), nil, nil)
	return nil
}

func (s *AuthService) ResendVerificationEmail(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	if user.EmailVerified {
		return nil
	}
	token, err := generateRandomToken(32)
	if err != nil {
		return err
	}
	if err := s.cache.SetEmailVerifyToken(ctx, token, user.ID, 24*time.Hour); err != nil {
		return fmt.Errorf("store email verify token: %w", err)
	}
	if s.emailSender != nil {
		return s.emailSender.SendVerificationEmail(ctx, user.Email, token)
	}
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	ok, err := verifyPassword(user.PasswordHash, oldPassword)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return ErrInvalidCredentials
	}
	if err := s.validatePassword(ctx, newPassword); err != nil {
		return err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	s.audit(ctx, &userID, "user.password_changed", "user", userID.String(), nil, nil)
	return nil
}

func (s *AuthService) createSession(ctx context.Context, userID uuid.UUID, ip, userAgent string) (*domain.UserSession, error) {
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

func (s *AuthService) isSettingEnabled(ctx context.Context, key string) bool {
	if s.settingsRepo == nil {
		return true
	}
	setting, err := s.settingsRepo.Get(ctx, key)
	if err != nil || setting == nil || setting.Value == "" {
		return true
	}
	return setting.Value != "false"
}

func registerCodeKey(email string) string {
	return "register_code:" + email
}

func (s *AuthService) validateEmailDomain(ctx context.Context, email string) error {
	if s.settingsRepo == nil {
		return nil
	}
	setting, err := s.settingsRepo.Get(ctx, "allowed_email_domains")
	if err != nil || setting.Value == "" {
		return nil
	}
	allowed := false
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 {
		domain := strings.ToLower(parts[1])
		for _, d := range strings.Split(setting.Value, ",") {
			d = strings.ToLower(strings.TrimSpace(d))
			if d != "" && d == domain {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		return fmt.Errorf("%w: email domain not allowed", ErrInvalidInput)
	}
	return nil
}

func (s *AuthService) validatePassword(ctx context.Context, password string) error {
	return ValidatePasswordByPolicy(password, ResolvePasswordPolicy(ctx, s.settingsRepo, s.cfg.Security))
}

func containsAny(s, chars string) bool {
	for _, c := range s {
		if strings.ContainsRune(chars, c) {
			return true
		}
	}
	return false
}

func (s *AuthService) audit(ctx context.Context, userID *uuid.UUID, action, resourceType, resourceID string, ip *string, details map[string]any) {
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
