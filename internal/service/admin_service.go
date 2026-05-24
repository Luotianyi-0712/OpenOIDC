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
	"strings"
	"time"

	"github.com/google/uuid"

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
}

func NewAdminService(
	userRepo port.UserRepository,
	providerCfgRepo port.ProviderConfigRepository,
	settingsRepo port.SettingsRepository,
	aliasRepo port.AliasRestrictionRepository,
	signingKeyRepo port.SigningKeyRepository,
	auditRepo port.AuditRepository,
) *AdminService {
	return &AdminService{
		userRepo:        userRepo,
		providerCfgRepo: providerCfgRepo,
		settingsRepo:    settingsRepo,
		aliasRepo:       aliasRepo,
		signingKeyRepo:  signingKeyRepo,
		auditRepo:       auditRepo,
	}
}

type AdminUserUpdate struct {
	DisplayName *string
	Status      *domain.UserStatus
	Role        *string
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
	if len(password) < 6 {
		return nil, fmt.Errorf("%w: password must be at least 6 characters", ErrInvalidInput)
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
	if updates.DisplayName != nil {
		user.DisplayName = *updates.DisplayName
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

func (s *AdminService) UpdateProvider(ctx context.Context, pc *domain.ProviderConfig) error {
	if !domain.IsValidProvider(pc.Provider) {
		return fmt.Errorf("%w: unknown provider", ErrInvalidInput)
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

func (s *AdminService) UpdateSetting(ctx context.Context, key, value, desc string) error {
	if key == "" {
		return fmt.Errorf("%w: key required", ErrInvalidInput)
	}
	return s.settingsRepo.Upsert(ctx, key, value, desc)
}

func (s *AdminService) ListSettings(ctx context.Context) ([]*domain.GlobalSetting, error) {
	return s.settingsRepo.List(ctx)
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
	if len(newPassword) < 6 {
		return fmt.Errorf("%w: password must be at least 6 characters", ErrInvalidInput)
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
