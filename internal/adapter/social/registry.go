package social

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/golang-jwt/jwt/v5"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type Registry struct {
	mu              sync.RWMutex
	providers       map[string]port.SocialProvider
	providerCfgRepo port.ProviderConfigRepository
}

func NewRegistry(providerCfgRepo port.ProviderConfigRepository) *Registry {
	return &Registry{
		providers:       make(map[string]port.SocialProvider),
		providerCfgRepo: providerCfgRepo,
	}
}

func (r *Registry) Register(p port.SocialProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (port.SocialProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("social provider %q not found", name)
	}
	return p, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.providers[name]
	return ok
}

func (r *Registry) Reload(ctx context.Context) error {
	if r.providerCfgRepo == nil {
		return nil
	}
	configs, err := r.providerCfgRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list provider configs: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove providers that are disabled in DB.
	for name := range r.providers {
		found := false
		for _, cfg := range configs {
			if cfg.Provider == name && cfg.IsEnabled {
				found = true
				break
			}
		}
		if !found {
			delete(r.providers, name)
		}
	}

	// Add/update providers from DB configs.
	for _, cfg := range configs {
		if !cfg.IsEnabled {
			continue
		}
		if cfg.ClientID == nil || *cfg.ClientID == "" {
			continue
		}
		secret := ""
		if cfg.ClientSecret != nil {
			secret = *cfg.ClientSecret
		}
		p := buildProvider(cfg.Provider, *cfg.ClientID, secret, cfg.ExtraConfig)
		if p != nil {
			r.providers[cfg.Provider] = p
		}
	}
	return nil
}

func buildProvider(name, clientID, clientSecret string, extra map[string]any) port.SocialProvider {
	switch name {
	case domain.ProviderGitHub:
		return NewGitHubProvider(clientID, clientSecret)
	case domain.ProviderGoogle:
		return NewGoogleProvider(clientID, clientSecret)
	case domain.ProviderGitLab:
		var baseURL string
		if extra != nil {
			baseURL, _ = extra["base_url"].(string)
		}
		return NewGitLabProvider(clientID, clientSecret, baseURL)
	case domain.ProviderGitee:
		return NewGiteeProvider(clientID, clientSecret)
	case domain.ProviderDiscord:
		return NewDiscordProvider(clientID, clientSecret)
	case domain.ProviderMicrosoft:
		var tenantID string
		if extra != nil {
			tenantID, _ = extra["tenant_id"].(string)
		}
		return NewMicrosoftProvider(clientID, clientSecret, tenantID)
	case domain.ProviderQQ:
		return NewQQProvider(clientID, clientSecret)
	case domain.ProviderWeChat:
		appID := clientID
		appSecret := clientSecret
		if extra != nil {
			if v, ok := extra["app_id"].(string); ok && v != "" {
				appID = v
			}
			if v, ok := extra["app_secret"].(string); ok && v != "" {
				appSecret = v
			}
		}
		return NewWeChatProvider(appID, appSecret)
	case domain.ProviderTelegram:
		return NewTelegramProvider(clientSecret)
	case domain.ProviderApple:
		if extra == nil {
			slog.Warn("apple provider missing extra config")
			return nil
		}
		teamID, _ := extra["team_id"].(string)
		keyID, _ := extra["key_id"].(string)
		privateKeyPEM, _ := extra["private_key"].(string)
		if teamID == "" || keyID == "" || privateKeyPEM == "" {
			slog.Warn("apple provider missing required fields (team_id, key_id, private_key)")
			return nil
		}
		privKey, err := jwt.ParseECPrivateKeyFromPEM([]byte(privateKeyPEM))
		if err != nil {
			slog.Warn("apple provider: failed to parse private key", "error", err)
			return nil
		}
		return NewAppleProvider(clientID, teamID, keyID, privKey)
	case domain.ProviderPhone:
		// Phone provider requires a PhoneCodeVerifier interface which is not
		// available at registry level. Phone provider is wired separately at
		// application startup, so we skip it here.
		return nil
	default:
		slog.Warn("unknown provider in DB config", "provider", name)
		return nil
	}
}

var _ port.SocialProviderRegistry = (*Registry)(nil)
