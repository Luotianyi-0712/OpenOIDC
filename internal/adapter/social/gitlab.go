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
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatar_url"`
	CreatedAt   string `json:"created_at"`
	ConfirmedAt string `json:"confirmed_at"`
}

func NewGitLabProvider(clientID, clientSecret, baseURL string, scopes []string) *OAuth2Provider {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	// Default scopes if not configured
	if len(scopes) == 0 {
		scopes = []string{"read_user"}
	}
	return &OAuth2Provider{
		name: domain.ProviderGitLab,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  baseURL + "/oauth/authorize",
				TokenURL: baseURL + "/oauth/token",
			},
			Scopes: scopes,
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
			if raw == nil {
				raw = map[string]any{}
			}
			emailVerified, emailVerificationKnown := gitlabEmailVerified(raw, u.ConfirmedAt)
			if emailVerificationKnown {
				raw["email_verified"] = emailVerified
			}
			raw = normalizeRawProfile(raw, u.Email)
			return &port.ProviderUserInfo{
				ProviderUID:   strconv.FormatInt(u.ID, 10),
				Email:         u.Email,
				EmailVerified: emailVerified,
				DisplayName:   display,
				AvatarURL:     u.AvatarURL,
				RawProfile:    raw,
			}, nil
		},
	}
}

func gitlabEmailVerified(raw map[string]any, confirmedAt string) (bool, bool) {
	for _, key := range []string{"email_verified", "verified_email", "email_confirmed"} {
		if value, ok := rawProfileBool(raw, key); ok {
			return value, true
		}
	}

	if strings.TrimSpace(confirmedAt) != "" {
		return true, true
	}
	if raw == nil {
		return false, false
	}
	value, exists := raw["confirmed_at"]
	if !exists {
		return false, false
	}
	switch v := value.(type) {
	case nil:
		return false, true
	case string:
		return strings.TrimSpace(v) != "", true
	default:
		return strings.TrimSpace(fmt.Sprint(v)) != "", true
	}
}
