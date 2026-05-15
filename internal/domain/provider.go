package domain

import (
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
	return false
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

type GlobalSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
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
