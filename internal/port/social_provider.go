package port

import (
	"context"
	"net/http"
)

type ProviderUserInfo struct {
	ProviderUID   string
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
	RawProfile    map[string]any
}

type SocialProvider interface {
	Name() string
	BeginAuth(ctx context.Context, state string, redirectURL string) (authURL string, err error)
	CompleteAuth(ctx context.Context, r *http.Request) (*ProviderUserInfo, error)
	SupportsRefresh() bool
	RefreshToken(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error)
}

type SocialProviderRegistry interface {
	Get(name string) (SocialProvider, error)
	List() []string
	IsEnabled(name string) bool
	Register(p SocialProvider)
	Reload(ctx context.Context) error
}
