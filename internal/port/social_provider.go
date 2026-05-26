package port

import (
	"context"
	"net/http"
	"time"
)

type ProviderTokenInfo struct {
	AccessToken  string
	RefreshToken string
	Expiry       *time.Time
	TokenType    string
	Scopes       []string
}

type ProviderUserInfo struct {
	ProviderUID   string
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
	RawProfile    map[string]any
	Token         *ProviderTokenInfo
}

type SocialProvider interface {
	Name() string
	BeginAuth(ctx context.Context, state string, redirectURL string) (authURL string, err error)
	CompleteAuth(ctx context.Context, r *http.Request) (*ProviderUserInfo, error)
	SupportsRefresh() bool
	RefreshToken(ctx context.Context, refreshToken string) (*ProviderTokenInfo, error)
}

type TokenValidatingProvider interface {
	ValidateToken(ctx context.Context, accessToken string) (*ProviderUserInfo, error)
}

type EnabledSocialProvider struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`
	LoginEnabled    bool   `json:"login_enabled"`
	RegisterEnabled bool   `json:"register_enabled"`
	SortOrder       int    `json:"-"`
}

type SocialProviderRegistry interface {
	Get(name string) (SocialProvider, error)
	List() []string
	ListPublic() []EnabledSocialProvider
	IsEnabled(name string) bool
	IsLoginEnabled(name string) bool
	IsRegisterEnabled(name string) bool
	Register(p SocialProvider)
	Reload(ctx context.Context) error
}
