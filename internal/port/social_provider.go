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

type SocialProviderRegistry interface {
	Get(name string) (SocialProvider, error)
	List() []string
	IsEnabled(name string) bool
	Register(p SocialProvider)
	Reload(ctx context.Context) error
}
