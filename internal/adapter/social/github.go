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
	ID          int64  `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatar_url"`
	CreatedAt   string `json:"created_at"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	PublicRepos int    `json:"public_repos"`
	PublicGists int    `json:"public_gists"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewGitHubProvider(clientID, clientSecret string, scopes []string) *OAuth2Provider {
	// Default scopes if not configured
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}
	return &OAuth2Provider{
		name: domain.ProviderGitHub,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       scopes,
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
	emailVerified := false
	emailVerificationKnown := false
	emailsBody, err := doGet(ctx, client, "https://api.github.com/user/emails")
	if err == nil {
		var emails []githubEmail
		if err := json.Unmarshal(emailsBody, &emails); err == nil {
			emailVerificationKnown = true
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					emailVerified = true
					break
				}
			}
			if email == "" {
				for _, e := range emails {
					if e.Verified {
						email = e.Email
						emailVerified = true
						break
					}
				}
			}
			if email != "" && !emailVerified {
				for _, e := range emails {
					if e.Email == email && e.Verified {
						emailVerified = true
						break
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
	if raw == nil {
		raw = map[string]any{}
	}
	if emailVerificationKnown {
		raw["email_verified"] = emailVerified
		raw["primary_email_verified"] = emailVerified
	}
	raw = normalizeRawProfile(raw, email)

	return &port.ProviderUserInfo{
		ProviderUID:   strconv.FormatInt(u.ID, 10),
		Email:         email,
		EmailVerified: emailVerified,
		DisplayName:   display,
		AvatarURL:     u.AvatarURL,
		RawProfile:    raw,
	}, nil
}
