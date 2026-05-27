package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type AdminService struct {
	userRepo        port.UserRepository
	providerCfgRepo port.ProviderConfigRepository
	settingsRepo    port.SettingsRepository
	aliasRepo       port.AliasRestrictionRepository
	signingKeyRepo  port.SigningKeyRepository
	auditRepo       port.AuditRepository
	passkeyRepo     port.PasskeyRepository
	securityCfg     config.SecurityConfig
}

func NewAdminService(
	userRepo port.UserRepository,
	providerCfgRepo port.ProviderConfigRepository,
	settingsRepo port.SettingsRepository,
	aliasRepo port.AliasRestrictionRepository,
	signingKeyRepo port.SigningKeyRepository,
	auditRepo port.AuditRepository,
	passkeyRepo port.PasskeyRepository,
	securityCfg config.SecurityConfig,
) *AdminService {
	return &AdminService{
		userRepo:        userRepo,
		providerCfgRepo: providerCfgRepo,
		settingsRepo:    settingsRepo,
		aliasRepo:       aliasRepo,
		signingKeyRepo:  signingKeyRepo,
		auditRepo:       auditRepo,
		passkeyRepo:     passkeyRepo,
		securityCfg:     securityCfg,
	}
}

type AdminUserUpdate struct {
	Email         *string
	EmailVerified *bool
	DisplayName   *string
	Alias         *string
	AvatarURL     *string
	Status        *domain.UserStatus
	Role          *string
}

func (s *AdminService) ListUsers(ctx context.Context, opts port.ListUsersOptions) ([]*domain.User, int64, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	return s.userRepo.List(ctx, opts)
}

func (s *AdminService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (s *AdminService) CreateUser(ctx context.Context, email, password, displayName, role string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if err := ValidatePasswordByPolicy(password, s.PasswordPolicy(ctx)); err != nil {
		return nil, err
	}

	switch role {
	case domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleUser:
		// valid
	default:
		return nil, fmt.Errorf("%w: role must be one of: user, admin, super_admin", ErrInvalidInput)
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
		Role:          role,
		Status:        domain.UserStatusActive,
		SecurityLevel: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	rt := "user"
	rid := user.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.user_created",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    now,
	})

	return user, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, id uuid.UUID, updates AdminUserUpdate) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("lookup user: %w", err)
	}
	if updates.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*updates.Email))
		if _, err := mail.ParseAddress(email); err != nil {
			return ErrInvalidEmail
		}
		if email != user.Email {
			existing, err := s.userRepo.GetByEmail(ctx, email)
			if err != nil && !errors.Is(err, port.ErrNotFound) {
				return fmt.Errorf("lookup email: %w", err)
			}
			if existing != nil && existing.ID != user.ID {
				return ErrAlreadyExists
			}
		}
		user.Email = email
	}
	if updates.EmailVerified != nil {
		user.EmailVerified = *updates.EmailVerified
	}
	if updates.DisplayName != nil {
		user.DisplayName = *updates.DisplayName
	}
	if updates.Alias != nil {
		alias := strings.TrimSpace(*updates.Alias)
		if alias == "" {
			user.Alias = nil
		} else {
			if user.Alias == nil || *user.Alias != alias {
				existing, err := s.userRepo.GetByAlias(ctx, alias)
				if err != nil && !errors.Is(err, port.ErrNotFound) {
					return fmt.Errorf("lookup alias: %w", err)
				}
				if existing != nil && existing.ID != user.ID {
					return ErrAlreadyExists
				}
			}
			user.Alias = &alias
		}
	}
	if updates.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*updates.AvatarURL)
	}
	if updates.Status != nil {
		user.Status = *updates.Status
	}
	if updates.Role != nil {
		user.Role = *updates.Role
	}
	user.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rt := "user"
	rid := id.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.user_updated",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rt := "user"
	rid := id.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.user_deleted",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *AdminService) ListUserPasskeys(ctx context.Context, userID uuid.UUID) ([]*domain.PasskeyCredential, error) {
	if _, err := s.GetUser(ctx, userID); err != nil {
		return nil, err
	}
	if s.passkeyRepo == nil {
		return []*domain.PasskeyCredential{}, nil
	}
	return s.passkeyRepo.ListByUser(ctx, userID)
}

func (s *AdminService) DeleteUserPasskey(ctx context.Context, userID, passkeyID uuid.UUID) error {
	if _, err := s.GetUser(ctx, userID); err != nil {
		return err
	}
	if s.passkeyRepo == nil {
		return ErrNotFound
	}
	creds, err := s.passkeyRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list passkeys: %w", err)
	}
	for _, cred := range creds {
		if cred.ID == passkeyID {
			if err := s.passkeyRepo.Delete(ctx, passkeyID); err != nil {
				return fmt.Errorf("delete passkey: %w", err)
			}
			rt := "passkey"
			rid := passkeyID.String()
			_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
				ID:           uuid.New(),
				Action:       "admin.user_passkey_deleted",
				ResourceType: &rt,
				ResourceID:   &rid,
				Details:      map[string]any{"user_id": userID.String()},
				CreatedAt:    time.Now().UTC(),
			})
			return nil
		}
	}
	return ErrNotFound
}

func (s *AdminService) OverrideSecurityLevel(ctx context.Context, id uuid.UUID, level int) error {
	if level < 0 {
		return fmt.Errorf("%w: level must be >= 0", ErrInvalidInput)
	}
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("lookup user: %w", err)
	}
	old := user.SecurityLevel
	if err := s.userRepo.UpdateSecurityLevel(ctx, id, level); err != nil {
		return fmt.Errorf("update security level: %w", err)
	}
	if err := s.auditRepo.CreateSecurityLevelChange(ctx, &domain.SecurityLevelChange{
		ID:        uuid.New(),
		UserID:    id,
		OldLevel:  old,
		NewLevel:  level,
		Reason:    "admin_override",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("record change: %w", err)
	}
	return nil
}

func (s *AdminService) ListProviders(ctx context.Context) ([]*domain.ProviderConfig, error) {
	return s.providerCfgRepo.List(ctx)
}

func (s *AdminService) GetProvider(ctx context.Context, provider string) (*domain.ProviderConfig, error) {
	p, err := s.providerCfgRepo.Get(ctx, provider)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *AdminService) CreateProvider(ctx context.Context, pc *domain.ProviderConfig) error {
	if pc == nil {
		return fmt.Errorf("%w: provider config required", ErrInvalidInput)
	}
	pc.Provider = strings.TrimSpace(pc.Provider)
	if !domain.IsValidCustomProviderKey(pc.Provider) {
		return fmt.Errorf("%w: provider must start with custom_ or oauth_ and contain only lowercase letters, numbers, _ or -", ErrInvalidInput)
	}
	if existing, err := s.providerCfgRepo.Get(ctx, pc.Provider); err != nil && !errors.Is(err, port.ErrNotFound) {
		return fmt.Errorf("lookup provider: %w", err)
	} else if existing != nil {
		return ErrAlreadyExists
	}
	pc.DisplayName = strings.TrimSpace(pc.DisplayName)
	if pc.DisplayName == "" {
		pc.DisplayName = pc.Provider
	}
	if pc.ExtraConfig == nil {
		pc.ExtraConfig = make(map[string]any)
	}
	pc.ExtraConfig["type"] = domain.ProviderTypeCustomOAuth2
	if err := validateCustomOAuth2ProviderConfig(pc); err != nil {
		return err
	}
	if pc.SortOrder == 0 {
		existing, err := s.providerCfgRepo.List(ctx)
		if err != nil {
			return fmt.Errorf("list providers: %w", err)
		}
		pc.SortOrder = len(existing) + 100
	}
	now := time.Now().UTC()
	pc.ID = uuid.New()
	pc.CreatedAt = now
	pc.UpdatedAt = now
	if err := s.providerCfgRepo.Upsert(ctx, pc); err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	rt := "provider_config"
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.provider_created",
		ResourceType: &rt,
		Details:      map[string]any{"provider": pc.Provider, "enabled": pc.IsEnabled},
		CreatedAt:    now,
	})
	return nil
}

func (s *AdminService) UpdateProvider(ctx context.Context, pc *domain.ProviderConfig) error {
	if !domain.IsValidProvider(pc.Provider) {
		return fmt.Errorf("%w: unknown provider", ErrInvalidInput)
	}
	if domain.IsValidCustomProviderKey(pc.Provider) || domain.IsCustomOAuth2Provider(pc) {
		if pc.ExtraConfig == nil {
			pc.ExtraConfig = make(map[string]any)
		}
		pc.ExtraConfig["type"] = domain.ProviderTypeCustomOAuth2
		if err := validateCustomOAuth2ProviderConfig(pc); err != nil {
			return err
		}
	}
	pc.UpdatedAt = time.Now().UTC()
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
		pc.CreatedAt = pc.UpdatedAt
	}
	if err := s.providerCfgRepo.Upsert(ctx, pc); err != nil {
		return fmt.Errorf("upsert provider: %w", err)
	}
	rt := "provider_config"
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.provider_updated",
		ResourceType: &rt,
		Details:      map[string]any{"provider": pc.Provider, "enabled": pc.IsEnabled},
		CreatedAt:    pc.UpdatedAt,
	})
	return nil
}

func (s *AdminService) DeleteProvider(ctx context.Context, provider string) error {
	provider = strings.TrimSpace(provider)
	if !domain.IsValidCustomProviderKey(provider) {
		return fmt.Errorf("%w: only custom OAuth2 providers can be deleted", ErrInvalidInput)
	}
	if _, err := s.providerCfgRepo.Get(ctx, provider); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("lookup provider: %w", err)
	}
	if err := s.providerCfgRepo.Delete(ctx, provider); err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	now := time.Now().UTC()
	rt := "provider_config"
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.provider_deleted",
		ResourceType: &rt,
		Details:      map[string]any{"provider": provider},
		CreatedAt:    now,
	})
	return nil
}

func validateCustomOAuth2ProviderConfig(pc *domain.ProviderConfig) error {
	if pc == nil || !pc.IsEnabled {
		return nil
	}
	if pc.ClientID == nil || strings.TrimSpace(*pc.ClientID) == "" {
		return fmt.Errorf("%w: client_id is required when custom OAuth2 provider is enabled", ErrInvalidInput)
	}
	if pc.ClientSecret == nil || strings.TrimSpace(*pc.ClientSecret) == "" {
		return fmt.Errorf("%w: client_secret is required when custom OAuth2 provider is enabled", ErrInvalidInput)
	}
	cfg := pc.CustomOAuth2Config()
	if customOAuth2ExtraString(pc.ExtraConfig, "user_id_field", "user_id_path") == "" {
		return fmt.Errorf("%w: user_id_field is required when custom OAuth2 provider is enabled", ErrInvalidInput)
	}
	for key, value := range map[string]string{
		"authorization_endpoint": cfg.AuthURL,
		"token_endpoint":         cfg.TokenURL,
		"userinfo_endpoint":      cfg.UserURL,
	} {
		if err := validateHTTPURL(key, value); err != nil {
			return err
		}
	}
	return nil
}

func customOAuth2ExtraString(extra map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, _ := extra[key].(string); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func validateHTTPURL(key, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: %s is required when custom OAuth2 provider is enabled", ErrInvalidInput, key)
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: %s must be a valid URL", ErrInvalidInput, key)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: %s must use http or https", ErrInvalidInput, key)
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return fmt.Errorf("%w: %s must use https unless it points to localhost", ErrInvalidInput, key)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "127.0.0.1" || host == "::1"
}

func (s *AdminService) GetSetting(ctx context.Context, key string) (*domain.GlobalSetting, error) {
	g, err := s.settingsRepo.Get(ctx, key)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return g, nil
}

func (s *AdminService) UpdateSetting(ctx context.Context, key, value, desc string, adminID uuid.UUID) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: key required", ErrInvalidInput)
	}
	if key == "site_url" {
		normalized, err := normalizeSiteURL(value)
		if err != nil {
			return err
		}
		value = normalized
	}
	if err := s.settingsRepo.Upsert(ctx, key, value, desc); err != nil {
		return err
	}
	if s.auditRepo != nil {
		rt := "setting"
		_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
			ID:           uuid.New(),
			UserID:       &adminID,
			Action:       "admin.setting_updated",
			ResourceType: &rt,
			ResourceID:   &key,
			Details:      map[string]any{"key": key},
			CreatedAt:    time.Now().UTC(),
		})
	}
	return nil
}

func normalizeSiteURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", fmt.Errorf("%w: site_url is required", ErrInvalidInput)
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%w: site_url must be a valid URL", ErrInvalidInput)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%w: site_url must use http or https", ErrInvalidInput)
	}
	return value, nil
}

func (s *AdminService) ListSettings(ctx context.Context) ([]*domain.GlobalSetting, error) {
	return s.settingsRepo.List(ctx)
}

func (s *AdminService) PasswordPolicy(ctx context.Context) PasswordPolicy {
	return ResolvePasswordPolicy(ctx, s.settingsRepo, s.securityCfg)
}

func (s *AdminService) CreateAliasRestriction(ctx context.Context, pattern, restrictionType, reason string) (*domain.AliasRestriction, error) {
	if pattern == "" {
		return nil, fmt.Errorf("%w: pattern required", ErrInvalidInput)
	}
	switch restrictionType {
	case "reserved", "blocked", "regex_blocked":
	default:
		return nil, fmt.Errorf("%w: restriction_type must be reserved|blocked|regex_blocked", ErrInvalidInput)
	}
	r := &domain.AliasRestriction{
		ID:              uuid.New(),
		Pattern:         pattern,
		RestrictionType: restrictionType,
		Reason:          reason,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.aliasRepo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("create restriction: %w", err)
	}
	return r, nil
}

func (s *AdminService) ListAliasRestrictions(ctx context.Context) ([]*domain.AliasRestriction, error) {
	return s.aliasRepo.List(ctx)
}

func (s *AdminService) DeleteAliasRestriction(ctx context.Context, id uuid.UUID) error {
	return s.aliasRepo.Delete(ctx, id)
}

func (s *AdminService) ListSigningKeys(ctx context.Context) ([]*domain.SigningKey, error) {
	return s.signingKeyRepo.List(ctx)
}

func (s *AdminService) GetCurrentSigningKey(ctx context.Context) (*domain.SigningKey, error) {
	k, err := s.signingKeyRepo.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return k, nil
}

func (s *AdminService) RotateSigningKey(ctx context.Context) (*domain.SigningKey, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(privKey)
	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	now := time.Now().UTC()
	keyID := uuid.NewString()
	newKey := &domain.SigningKey{
		ID:         uuid.New(),
		KeyID:      keyID,
		Algorithm:  "RS256",
		PrivateKey: privPEM,
		PublicKey:  pubPEM,
		IsCurrent:  true,
		CreatedAt:  now,
	}

	old, err := s.signingKeyRepo.GetCurrent(ctx)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, fmt.Errorf("get current key: %w", err)
	}
	if err := s.signingKeyRepo.Create(ctx, newKey); err != nil {
		return nil, fmt.Errorf("create signing key: %w", err)
	}
	if old != nil {
		if err := s.signingKeyRepo.Rotate(ctx, old.ID, newKey.ID); err != nil {
			return nil, fmt.Errorf("rotate: %w", err)
		}
	}
	rt := "signing_key"
	rid := newKey.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.signing_key_rotated",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    now,
	})
	return newKey, nil
}

func (s *AdminService) ListAuditLogs(ctx context.Context, opts port.ListAuditOptions) ([]*domain.AuditLog, int64, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	return s.auditRepo.ListLogs(ctx, opts)
}

// ResetUserPassword allows an admin to force-reset a user's password.
func (s *AdminService) ResetUserPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	if err := ValidatePasswordByPolicy(newPassword, s.PasswordPolicy(ctx)); err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("lookup user: %w", err)
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rt := "user"
	rid := userID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		Action:       "admin.user_password_reset",
		ResourceType: &rt,
		ResourceID:   &rid,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}
