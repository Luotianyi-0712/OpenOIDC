package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropic/oidc-platform/internal/adapter/social"
	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

type guildSyncBindingRepo struct {
	port.BindingRepository
	binding *domain.SocialBinding
	updates int
}

func (r *guildSyncBindingRepo) Update(_ context.Context, b *domain.SocialBinding) error {
	r.binding = b
	r.updates++
	return nil
}
func (r *guildSyncBindingRepo) ListDueAuthChecks(context.Context, time.Time, int) ([]*domain.SocialBinding, error) {
	return []*domain.SocialBinding{r.binding}, nil
}
func (r *guildSyncBindingRepo) ListByUser(context.Context, uuid.UUID) ([]*domain.SocialBinding, error) {
	return []*domain.SocialBinding{r.binding}, nil
}

type guildSyncUserRepo struct {
	port.UserRepository
	user *domain.User
}

func (r *guildSyncUserRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return r.user, nil
}
func (r *guildSyncUserRepo) UpdateSecurityLevel(_ context.Context, _ uuid.UUID, level int) error {
	r.user.SecurityLevel = level
	return nil
}

type guildSyncRuleRepo struct {
	port.RuleRepository
	rule *domain.SecurityLevelRule
}

func (r *guildSyncRuleRepo) ListActive(context.Context) ([]*domain.SecurityLevelRule, error) {
	return []*domain.SecurityLevelRule{r.rule}, nil
}

type guildSyncAuditRepo struct{ port.AuditRepository }

func (*guildSyncAuditRepo) CreateLog(context.Context, *domain.AuditLog) error { return nil }
func (*guildSyncAuditRepo) CreateSecurityLevelChange(context.Context, *domain.SecurityLevelChange) error {
	return nil
}

type guildSyncProvider struct {
	port.SocialProvider
	refreshErr, validateErr error
	profile                 map[string]any
	validatedToken          string
}

func (*guildSyncProvider) Name() string          { return domain.ProviderDiscord }
func (*guildSyncProvider) SupportsRefresh() bool { return true }
func (p *guildSyncProvider) RefreshToken(context.Context, string) (*port.ProviderTokenInfo, error) {
	if p.refreshErr != nil {
		return nil, p.refreshErr
	}
	return &port.ProviderTokenInfo{AccessToken: "new-access", RefreshToken: "rotated-refresh", Scopes: []string{"identify", "guilds.members.read"}}, nil
}
func (p *guildSyncProvider) ValidateToken(_ context.Context, token string) (*port.ProviderUserInfo, error) {
	p.validatedToken = token
	if p.validateErr != nil {
		return nil, p.validateErr
	}
	return &port.ProviderUserInfo{ProviderUID: "discord-user", RawProfile: p.profile}, nil
}

func TestDiscordSyncRefreshesMembershipAndLevel(t *testing.T) {
	const guildID = "123456789012345678"
	for _, tc := range []struct {
		name                    string
		entry                   map[string]any
		refreshErr, validateErr error
		level                   int
	}{
		{"left guild", map[string]any{"is_member": false}, nil, nil, 0},
		{"missing scope", map[string]any{"status": "unknown"}, nil, nil, 0},
		{"targets removed", nil, nil, nil, 0},
		{"user fetch failed", nil, nil, errors.New("network unavailable"), 0},
		{"token refresh failed", nil, errors.New("refresh unavailable"), nil, 0},
		{"still a member", map[string]any{"is_member": true, "joined_at": "2020-01-01T00:00:00Z"}, nil, nil, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := &domain.User{ID: uuid.New(), SecurityLevel: 2, Status: domain.UserStatusActive}
			access, refresh := "old-access", "old-refresh"
			binding := &domain.SocialBinding{ID: uuid.New(), UserID: user.ID, Provider: domain.ProviderDiscord, ProviderUID: "discord-user", Status: domain.SocialBindingStatusActive, AccessToken: &access, RefreshToken: &refresh, RawProfile: map[string]any{"guilds": map[string]any{guildID: map[string]any{"is_member": true, "joined_at": "2020-01-01T00:00:00Z"}}}}
			bindings := &guildSyncBindingRepo{binding: binding}
			rules := &guildSyncRuleRepo{rule: &domain.SecurityLevelRule{ID: uuid.New(), Level: 2, Conditions: domain.RuleConditions{Conditions: []domain.RuleCondition{{Type: domain.ConditionDiscordGuildAgeDays, Field: guildID, MinDays: 60}}}}}
			audit := &guildSyncAuditRepo{}
			security := NewSecurityLevelService(rules, bindings, &guildSyncUserRepo{user: user}, audit)
			guilds := map[string]any{}
			if tc.entry != nil {
				guilds[guildID] = tc.entry
			}
			provider := &guildSyncProvider{refreshErr: tc.refreshErr, validateErr: tc.validateErr, profile: map[string]any{"guilds": guilds}}
			registry := social.NewRegistry(nil)
			registry.Register(provider)
			svc := &SocialService{registry: registry, bindingRepo: bindings, securitySvc: security, auditRepo: audit}
			if err := svc.SyncAuthorizationStatus(context.Background(), 100, time.Now()); err != nil {
				t.Fatal(err)
			}
			if user.SecurityLevel != tc.level {
				t.Fatalf("level=%d, want %d", user.SecurityLevel, tc.level)
			}
			if bindings.updates != 1 {
				t.Fatalf("snapshot was not persisted: %d updates", bindings.updates)
			}
			if tc.refreshErr == nil && provider.validatedToken != "new-access" {
				t.Fatal("profile was not fetched with refreshed token")
			}
			if tc.refreshErr == nil && *bindings.binding.RefreshToken != "rotated-refresh" {
				t.Fatal("rotated refresh token was lost")
			}
			if tc.refreshErr != nil || tc.validateErr != nil {
				if _, present := bindings.binding.RawProfile["guilds"]; present {
					t.Fatal("failed refresh retained stale membership")
				}
			}
		})
	}
}
