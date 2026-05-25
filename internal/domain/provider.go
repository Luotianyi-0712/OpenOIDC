package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ProviderGitHub    = "github"
	ProviderGoogle    = "google"
	ProviderGitLab    = "gitlab"
	ProviderGitee     = "gitee"
	ProviderDiscord   = "discord"
	ProviderTelegram  = "telegram"
	ProviderMicrosoft = "microsoft"
	ProviderApple     = "apple"
	ProviderQQ        = "qq"
	ProviderWeChat    = "wechat"
	ProviderPhone     = "phone"

	ProviderTypeBuiltIn      = "built_in"
	ProviderTypeCustomOAuth2 = "custom_oauth2"
)

func AllProviders() []string {
	return []string{
		ProviderGitHub,
		ProviderGoogle,
		ProviderGitLab,
		ProviderGitee,
		ProviderDiscord,
		ProviderTelegram,
		ProviderMicrosoft,
		ProviderApple,
		ProviderQQ,
		ProviderWeChat,
		ProviderPhone,
	}
}

func IsValidProvider(name string) bool {
	for _, p := range AllProviders() {
		if p == name {
			return true
		}
	}
	return IsValidCustomProviderKey(name)
}

func IsBuiltInProvider(name string) bool {
	for _, p := range AllProviders() {
		if p == name {
			return true
		}
	}
	return false
}

func IsValidCustomProviderKey(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 3 || len(name) > 60 {
		return false
	}
	if IsBuiltInProvider(name) {
		return false
	}
	if !(strings.HasPrefix(name, "custom_") || strings.HasPrefix(name, "oauth_")) {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

type ProviderConfig struct {
	ID           uuid.UUID      `json:"id"`
	Provider     string         `json:"provider"`
	DisplayName  string         `json:"display_name"`
	IsEnabled    bool           `json:"is_enabled"`
	ClientID     *string        `json:"client_id,omitempty"`
	ClientSecret *string        `json:"client_secret,omitempty"`
	ExtraConfig  map[string]any `json:"extra_config,omitempty"`
	SortOrder    int            `json:"sort_order"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CustomOAuth2Config struct {
	AuthURL    string
	TokenURL   string
	UserURL    string
	Scopes     []string
	IDPath     string
	EmailPath  string
	NamePath   string
	AvatarPath string
}

func ProviderType(pc *ProviderConfig) string {
	if pc == nil {
		return ProviderTypeBuiltIn
	}
	if pc.ExtraConfig != nil {
		if typ, _ := pc.ExtraConfig["type"].(string); typ != "" {
			return typ
		}
		if typ, _ := pc.ExtraConfig["provider_type"].(string); typ != "" {
			return typ
		}
	}
	if IsBuiltInProvider(pc.Provider) {
		return ProviderTypeBuiltIn
	}
	return ProviderTypeCustomOAuth2
}

func IsCustomOAuth2Provider(pc *ProviderConfig) bool {
	return ProviderType(pc) == ProviderTypeCustomOAuth2
}

func (pc *ProviderConfig) CustomOAuth2Config() CustomOAuth2Config {
	cfg := CustomOAuth2Config{
		IDPath:     "id",
		EmailPath:  "email",
		NamePath:   "name",
		AvatarPath: "avatar_url",
	}
	if pc == nil || pc.ExtraConfig == nil {
		return cfg
	}
	cfg.AuthURL = firstExtraString(pc.ExtraConfig, "authorization_endpoint", "auth_url")
	cfg.TokenURL = firstExtraString(pc.ExtraConfig, "token_endpoint", "token_url")
	cfg.UserURL = firstExtraString(pc.ExtraConfig, "userinfo_endpoint", "userinfo_url", "user_url")
	cfg.Scopes = extraStringSlice(pc.ExtraConfig, "scopes")
	if v := firstExtraString(pc.ExtraConfig, "user_id_field", "user_id_path"); v != "" {
		cfg.IDPath = v
	}
	if v := firstExtraString(pc.ExtraConfig, "email_field", "email_path"); v != "" {
		cfg.EmailPath = v
	}
	if v := firstExtraString(pc.ExtraConfig, "name_field", "name_path"); v != "" {
		cfg.NamePath = v
	}
	if v := firstExtraString(pc.ExtraConfig, "avatar_field", "avatar_path"); v != "" {
		cfg.AvatarPath = v
	}
	return cfg
}

type GlobalSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func extraString(extra map[string]any, key string) string {
	v, _ := extra[key].(string)
	return strings.TrimSpace(v)
}

func firstExtraString(extra map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := extraString(extra, key); value != "" {
			return value
		}
	}
	return ""
}

func extraStringSlice(extra map[string]any, key string) []string {
	v, ok := extra[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return cleanStringSlice(t)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return cleanStringSlice(out)
	case string:
		parts := strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
		return cleanStringSlice(parts)
	default:
		return nil
	}
}

func cleanStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

type AliasRestriction struct {
	ID              uuid.UUID `json:"id"`
	Pattern         string    `json:"pattern"`
	RestrictionType string    `json:"restriction_type"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           uuid.UUID      `json:"id"`
	UserID       *uuid.UUID     `json:"user_id"`
	Action       string         `json:"action"`
	ResourceType *string        `json:"resource_type"`
	ResourceID   *string        `json:"resource_id"`
	IPAddress    *string        `json:"ip_address"`
	UserAgent    *string        `json:"user_agent"`
	Details      map[string]any `json:"details"`
	CreatedAt    time.Time      `json:"created_at"`
}

type SigningKey struct {
	ID         uuid.UUID  `json:"id"`
	KeyID      string     `json:"key_id"`
	Algorithm  string     `json:"algorithm"`
	PrivateKey []byte     `json:"-"`
	PublicKey  []byte     `json:"public_key"`
	IsCurrent  bool       `json:"is_current"`
	CreatedAt  time.Time  `json:"created_at"`
	RotatedAt  *time.Time `json:"rotated_at"`
}
