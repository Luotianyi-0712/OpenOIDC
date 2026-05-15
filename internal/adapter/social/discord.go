package social

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type discordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
}

func NewDiscordProvider(clientID, clientSecret string) *OAuth2Provider {
	return &OAuth2Provider{
		name: domain.ProviderDiscord,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
			Scopes: []string{"identify", "email"},
		},
		userURL: "https://discord.com/api/users/@me",
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var u discordUser
			if err := json.Unmarshal(body, &u); err != nil {
				return nil, fmt.Errorf("decode discord user: %w", err)
			}
			display := u.GlobalName
			if display == "" {
				display = u.Username
			}
			avatar := ""
			if u.Avatar != "" {
				ext := "png"
				if len(u.Avatar) >= 2 && u.Avatar[:2] == "a_" {
					ext = "gif"
				}
				avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", u.ID, u.Avatar, ext)
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			return &port.ProviderUserInfo{
				ProviderUID: u.ID,
				Email:       u.Email,
				DisplayName: display,
				AvatarURL:   avatar,
				RawProfile:  raw,
			}, nil
		},
	}
}
