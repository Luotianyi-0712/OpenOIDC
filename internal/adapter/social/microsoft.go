package social

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type microsoftUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

func NewMicrosoftProvider(clientID, clientSecret, tenantID string) *OAuth2Provider {
	if tenantID == "" {
		tenantID = "common"
	}
	return &OAuth2Provider{
		name: domain.ProviderMicrosoft,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/" + tenantID + "/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/" + tenantID + "/oauth2/v2.0/token",
			},
			Scopes: []string{"openid", "profile", "email", "User.Read", "offline_access"},
		},
		userURL: "https://graph.microsoft.com/v1.0/me",
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var u microsoftUser
			if err := json.Unmarshal(body, &u); err != nil {
				return nil, fmt.Errorf("decode microsoft user: %w", err)
			}
			email := u.Mail
			if email == "" {
				email = u.UserPrincipalName
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			return &port.ProviderUserInfo{
				ProviderUID: u.ID,
				Email:       email,
				DisplayName: u.DisplayName,
				RawProfile:  raw,
			}, nil
		},
	}
}
