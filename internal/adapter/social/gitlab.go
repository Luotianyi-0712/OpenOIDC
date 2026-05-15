package social

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type gitlabUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func NewGitLabProvider(clientID, clientSecret, baseURL string) *OAuth2Provider {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &OAuth2Provider{
		name: domain.ProviderGitLab,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  baseURL + "/oauth/authorize",
				TokenURL: baseURL + "/oauth/token",
			},
			Scopes: []string{"read_user"},
		},
		userURL: baseURL + "/api/v4/user",
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var u gitlabUser
			if err := json.Unmarshal(body, &u); err != nil {
				return nil, fmt.Errorf("decode gitlab user: %w", err)
			}
			display := u.Name
			if display == "" {
				display = u.Username
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			return &port.ProviderUserInfo{
				ProviderUID: strconv.FormatInt(u.ID, 10),
				Email:       u.Email,
				DisplayName: display,
				AvatarURL:   u.AvatarURL,
				RawProfile:  raw,
			}, nil
		},
	}
}
