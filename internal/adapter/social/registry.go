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
	publicProviders map[string]port.EnabledSocialProvider
	providerCfgRepo port.ProviderConfigRepository
}

func NewRegistry(providerCfgRepo port.ProviderConfigRepository) *Registry {
	return &Registry{
		providers:       make(map[string]port.SocialProvider),
		publicProviders: make(map[string]port.EnabledSocialProvider),
		providerCfgRepo: providerCfgRepo,
	}
}

func (r *Registry) Register(p port.SocialProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
	if _, ok := r.publicProviders[p.Name()]; !ok {
		r.publicProviders[p.Name()] = port.EnabledSocialProvider{
			Name:            p.Name(),
			DisplayName:     providerDisplayName(p.Name()),
			Type:            domain.ProviderTypeBuiltIn,
			LoginEnabled:    true,
			RegisterEnabled: true,
		}
	}
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

func (r *Registry) ListPublic() []port.EnabledSocialProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	providers := make([]port.EnabledSocialProvider, 0, len(r.publicProviders))
	for _, p := range r.publicProviders {
		if _, ok := r.providers[p.Name]; ok {
			providers = append(providers, p)
		}
	}
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].SortOrder == providers[j].SortOrder {
			return providers[i].Name < providers[j].Name
		}
		return providers[i].SortOrder < providers[j].SortOrder
	})
	return providers
}

func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.providers[name]
	return ok
}

func (r *Registry) IsLoginEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.publicProviders[name]
	return ok && p.LoginEnabled
}

func (r *Registry) IsRegisterEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.publicProviders[name]
	return ok && p.RegisterEnabled
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
			delete(r.publicProviders, name)
		}
	}

	// Add/update providers from DB configs.
	for _, cfg := range configs {
		if !cfg.IsEnabled {
			delete(r.providers, cfg.Provider)
			delete(r.publicProviders, cfg.Provider)
			continue
		}
		if cfg.ClientID == nil || *cfg.ClientID == "" {
			continue
		}
		secret := ""
		if cfg.ClientSecret != nil {
			secret = *cfg.ClientSecret
		}
		p := buildProvider(cfg, *cfg.ClientID, secret)
		if p != nil {
			r.providers[cfg.Provider] = p
			r.publicProviders[cfg.Provider] = providerPublicInfo(cfg)
		}
	}
	return nil
}

func buildProvider(cfg *domain.ProviderConfig, clientID, clientSecret string) port.SocialProvider {
	name := cfg.Provider
	extra := cfg.ExtraConfig
	if domain.IsCustomOAuth2Provider(cfg) {
		return NewCustomOAuth2Provider(name, clientID, clientSecret, cfg.CustomOAuth2Config())
	}

	switch name {
	case domain.ProviderGitHub:
		return NewGitHubProvider(clientID, clientSecret, cfg.Scopes)
	case domain.ProviderGoogle:
		return NewGoogleProvider(clientID, clientSecret, cfg.Scopes)
	case domain.ProviderGitLab:
		var baseURL string
		if extra != nil {
			baseURL, _ = extra["base_url"].(string)
		}
		return NewGitLabProvider(clientID, clientSecret, baseURL, cfg.Scopes)
	case domain.ProviderGitee:
		return NewGiteeProvider(clientID, clientSecret, cfg.Scopes)
	case domain.ProviderLinuxDO:
		return NewLinuxDOProvider(clientID, clientSecret, cfg.Scopes)
	case domain.ProviderDiscord:
		return NewDiscordProvider(clientID, clientSecret, cfg.Scopes)
	case domain.ProviderMicrosoft:
		var tenantID string
		if extra != nil {
			tenantID, _ = extra["tenant_id"].(string)
		}
		return NewMicrosoftProvider(clientID, clientSecret, tenantID, cfg.Scopes)
	case domain.ProviderQQ:
		return NewQQProvider(clientID, clientSecret, cfg.Scopes)
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
		return NewAppleProvider(clientID, teamID, keyID, privKey, cfg.Scopes)
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

func providerPublicInfo(cfg *domain.ProviderConfig) port.EnabledSocialProvider {
	name := cfg.Provider
	displayName := cfg.DisplayName
	if displayName == "" {
		displayName = providerDisplayName(name)
	}
	info := port.EnabledSocialProvider{
		Name:            name,
		DisplayName:     displayName,
		Type:            domain.ProviderType(cfg),
		LoginEnabled:    true,
		RegisterEnabled: true,
		SortOrder:       cfg.SortOrder,
	}
	if cfg.ExtraConfig != nil {
		if iconURL, _ := cfg.ExtraConfig["icon_url"].(string); iconURL != "" {
			info.IconURL = iconURL
		}
		if v, ok := cfg.ExtraConfig["login_enabled"].(bool); ok {
			info.LoginEnabled = v
		}
		if v, ok := cfg.ExtraConfig["register_enabled"].(bool); ok {
			info.RegisterEnabled = v
		}
	}
	return info
}

func providerDisplayName(name string) string {
	switch name {
	case domain.ProviderGitHub:
		return "GitHub"
	case domain.ProviderGoogle:
		return "Google"
	case domain.ProviderGitLab:
		return "GitLab"
	case domain.ProviderGitee:
		return "Gitee"
	case domain.ProviderLinuxDO:
		return "Linux DO"
	case domain.ProviderDiscord:
		return "Discord"
	case domain.ProviderTelegram:
		return "Telegram"
	case domain.ProviderMicrosoft:
		return "Microsoft"
	case domain.ProviderApple:
		return "Apple"
	case domain.ProviderQQ:
		return "QQ"
	case domain.ProviderWeChat:
		return "WeChat"
	case domain.ProviderPhone:
		return "Phone"
	default:
		return name
	}
}

var _ port.SocialProviderRegistry = (*Registry)(nil)
