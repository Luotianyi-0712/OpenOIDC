package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewGitHubProvider(clientID, clientSecret string) *OAuth2Provider {
	return &OAuth2Provider{
		name: domain.ProviderGitHub,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
		userURL:   "https://api.github.com/user",
		fetchUser: fetchGitHubUser,
	}
}

func fetchGitHubUser(ctx context.Context, client *http.Client, _ *oauth2.Token) (*port.ProviderUserInfo, error) {
	body, err := doGet(ctx, client, "https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("fetch github user: %w", err)
	}
	var u githubUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("decode github user: %w", err)
	}

	email := u.Email
	if email == "" {
		emailsBody, err := doGet(ctx, client, "https://api.github.com/user/emails")
		if err == nil {
			var emails []githubEmail
			if err := json.Unmarshal(emailsBody, &emails); err == nil {
				for _, e := range emails {
					if e.Primary && e.Verified {
						email = e.Email
						break
					}
				}
				if email == "" {
					for _, e := range emails {
						if e.Verified {
							email = e.Email
							break
						}
					}
				}
			}
		}
	}

	display := u.Name
	if display == "" {
		display = u.Login
	}

	var raw map[string]any
	_ = json.Unmarshal(body, &raw)

	return &port.ProviderUserInfo{
		ProviderUID:   strconv.FormatInt(u.ID, 10),
		Email:         email,
		EmailVerified: email != "",
		DisplayName:   display,
		AvatarURL:     u.AvatarURL,
		RawProfile:    raw,
	}, nil
}
