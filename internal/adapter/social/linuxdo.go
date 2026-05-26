package social

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

const (
	linuxDOAuthURL  = "https://connect.linux.do/oauth2/authorize"
	linuxDOTokenURL = "https://connect.linux.do/oauth2/token"
	linuxDOUserURL  = "https://connect.linux.do/api/user"
	linuxDOBaseURL  = "https://linux.do"
)

func NewLinuxDOProvider(clientID, clientSecret string) *OAuth2Provider {
	return &OAuth2Provider{
		name: domain.ProviderLinuxDO,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:   linuxDOAuthURL,
				TokenURL:  linuxDOTokenURL,
				AuthStyle: oauth2.AuthStyleInParams,
			},
			Scopes: []string{"user"},
		},
		userURL:   linuxDOUserURL,
		parseUser: parseLinuxDOUser,
	}
}

func parseLinuxDOUser(body []byte) (*port.ProviderUserInfo, error) {
	var raw map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode linux do user: %w", err)
	}

	uid := valueAtPath(raw, "id")
	if uid == "" {
		return nil, fmt.Errorf("linux do user missing id")
	}

	email := valueAtPath(raw, "email")
	emailVerified, _ := rawProfileBool(raw, "email_verified")
	username := valueAtPath(raw, "username")
	display := valueAtPath(raw, "name")
	if display == "" {
		display = username
	}
	if display == "" {
		display = "Linux DO #" + uid
	}

	avatarURL := normalizeLinuxDOAvatarURL(firstNonEmpty(
		valueAtPath(raw, "avatar_template"),
		valueAtPath(raw, "avatar_url"),
		valueAtPath(raw, "avatar"),
	))
	if avatarURL != "" {
		raw["avatar_url"] = avatarURL
	}
	raw = normalizeRawProfile(raw, email)

	return &port.ProviderUserInfo{
		ProviderUID:   uid,
		Email:         email,
		EmailVerified: emailVerified,
		DisplayName:   display,
		AvatarURL:     avatarURL,
		RawProfile:    raw,
	}, nil
}

func normalizeLinuxDOAvatarURL(template string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	template = strings.ReplaceAll(template, "{size}", "96")
	if strings.HasPrefix(template, "//") {
		return "https:" + template
	}
	u, err := url.Parse(template)
	if err == nil && u.IsAbs() {
		return template
	}
	if strings.HasPrefix(template, "/") {
		return linuxDOBaseURL + template
	}
	return template
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
