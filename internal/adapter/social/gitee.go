package social

import (
	"encoding/json"
	"fmt"
	"strconv"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type giteeUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func NewGiteeProvider(clientID, clientSecret string, scopes []string) *OAuth2Provider {
	// Default scopes if not configured
	if len(scopes) == 0 {
		scopes = []string{"user_info"}
	}
	return &OAuth2Provider{
		name: domain.ProviderGitee,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://gitee.com/oauth/authorize",
				TokenURL: "https://gitee.com/oauth/token",
			},
			Scopes: scopes,
		},
		userURL: "https://gitee.com/api/v5/user",
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var u giteeUser
			if err := json.Unmarshal(body, &u); err != nil {
				return nil, fmt.Errorf("decode gitee user: %w", err)
			}
			display := u.Name
			if display == "" {
				display = u.Login
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			raw = normalizeRawProfile(raw, u.Email)
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
