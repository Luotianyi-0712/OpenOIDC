package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
)

type ListUsersOptions struct {
	Search string
	Status *domain.UserStatus
	Offset int
	Limit  int
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByAlias(ctx context.Context, alias string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateSecurityLevel(ctx context.Context, id uuid.UUID, level int) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	List(ctx context.Context, opts ListUsersOptions) ([]*domain.User, int64, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type BindingRepository interface {
	Create(ctx context.Context, b *domain.SocialBinding) error
	GetByProviderUID(ctx context.Context, provider, uid string) (*domain.SocialBinding, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.SocialBinding, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.SocialBinding, error)
	Delete(ctx context.Context, userID uuid.UUID, provider string) error
	UpdateTokens(ctx context.Context, id uuid.UUID, access, refresh string, expiry *time.Time) error
}

type ClientRepository interface {
	Create(ctx context.Context, c *domain.OIDCClient) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.OIDCClient, error)
	GetByClientID(ctx context.Context, clientID string) (*domain.OIDCClient, error)
	List(ctx context.Context, offset, limit int) ([]*domain.OIDCClient, int64, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID, offset, limit int) ([]*domain.OIDCClient, int64, error)
	Update(ctx context.Context, c *domain.OIDCClient) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateSecret(ctx context.Context, id uuid.UUID, hash, plain string) error
}

type ClientAccessRuleRepository interface {
	Create(ctx context.Context, r *domain.ClientAccessRule) error
	ListByClient(ctx context.Context, clientID uuid.UUID) ([]*domain.ClientAccessRule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type RuleRepository interface {
	Create(ctx context.Context, r *domain.SecurityLevelRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SecurityLevelRule, error)
	ListActive(ctx context.Context) ([]*domain.SecurityLevelRule, error)
	ListAll(ctx context.Context) ([]*domain.SecurityLevelRule, error)
	Update(ctx context.Context, r *domain.SecurityLevelRule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type SessionRepository interface {
	Create(ctx context.Context, s *domain.UserSession) error
	GetByToken(ctx context.Context, token string) (*domain.UserSession, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserSession, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	CountActive(ctx context.Context) (int64, error)
}

type ProviderConfigRepository interface {
	Get(ctx context.Context, provider string) (*domain.ProviderConfig, error)
	List(ctx context.Context) ([]*domain.ProviderConfig, error)
	Upsert(ctx context.Context, pc *domain.ProviderConfig) error
}

type ListAuditOptions struct {
	UserID *uuid.UUID
	Action string
	Offset int
	Limit  int
}

type AuditRepository interface {
	CreateLog(ctx context.Context, log *domain.AuditLog) error
	ListLogs(ctx context.Context, opts ListAuditOptions) ([]*domain.AuditLog, int64, error)
	CreateSecurityLevelChange(ctx context.Context, c *domain.SecurityLevelChange) error
	ListSecurityLevelChanges(ctx context.Context, userID uuid.UUID) ([]*domain.SecurityLevelChange, error)
}

type SettingsRepository interface {
	Get(ctx context.Context, key string) (*domain.GlobalSetting, error)
	Upsert(ctx context.Context, key, value, desc string) error
	List(ctx context.Context) ([]*domain.GlobalSetting, error)
}

type AliasRestrictionRepository interface {
	Create(ctx context.Context, r *domain.AliasRestriction) error
	List(ctx context.Context) ([]*domain.AliasRestriction, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type SigningKeyRepository interface {
	Create(ctx context.Context, k *domain.SigningKey) error
	GetCurrent(ctx context.Context) (*domain.SigningKey, error)
	List(ctx context.Context) ([]*domain.SigningKey, error)
	Rotate(ctx context.Context, oldID, newID uuid.UUID) error
}

type RiskReportRepository interface {
	Create(ctx context.Context, r *domain.RiskReport) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.RiskReport, error)
	ListByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.RiskReport, error)
	ListByClient(ctx context.Context, clientID uuid.UUID, offset, limit int) ([]*domain.RiskReport, int64, error)
	ListPending(ctx context.Context, offset, limit int) ([]*domain.RiskReport, int64, error)
	Update(ctx context.Context, r *domain.RiskReport) error
	CountConfirmedByTarget(ctx context.Context, targetID uuid.UUID) (int64, error)
}

type RiskListRepository interface {
	Add(ctx context.Context, entry *domain.RiskListEntry) error
	Check(ctx context.Context, provider, providerUID string) (*domain.RiskListEntry, error)
	List(ctx context.Context, offset, limit int) ([]*domain.RiskListEntry, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ConsentRepository provides read access to OAuth2 session/consent data.
type ConsentRepository interface {
	// ListAuthorizedApps returns apps a user has authorized (based on oauth2_sessions).
	ListAuthorizedApps(ctx context.Context, userID uuid.UUID) ([]*domain.UserAuthorization, error)
	// CountUniqueUsers returns the number of unique users who have authorized a given client.
	CountUniqueUsers(ctx context.Context, clientID string) (int64, error)
	// DeleteByUserAndClient revokes a user's authorization to a specific client.
	DeleteByUserAndClient(ctx context.Context, userID uuid.UUID, clientID string) error
}
