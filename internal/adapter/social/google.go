package social

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type googleUser struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func NewGoogleProvider(clientID, clientSecret string) *OAuth2Provider {
	return &OAuth2Provider{
		name: domain.ProviderGoogle,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "profile", "email"},
		},
		userURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		authOptions: []oauth2.AuthCodeOption{
			oauth2.AccessTypeOffline,
		},
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var u googleUser
			if err := json.Unmarshal(body, &u); err != nil {
				return nil, fmt.Errorf("decode google user: %w", err)
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			return &port.ProviderUserInfo{
				ProviderUID: u.ID,
				Email:       u.Email,
				DisplayName: u.Name,
				AvatarURL:   u.Picture,
				RawProfile:  raw,
			}, nil
		},
	}
}
