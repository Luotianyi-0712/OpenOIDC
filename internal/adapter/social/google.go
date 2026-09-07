package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type googleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	HostedDomain  string `json:"hd"`
}

type googlePeoplePhoneMetadata struct {
	Primary  bool `json:"primary"`
	Verified bool `json:"verified"`
}

type googlePeoplePhone struct {
	Value         string                    `json:"value"`
	CanonicalForm string                    `json:"canonicalForm"`
	Type          string                    `json:"type"`
	Metadata      googlePeoplePhoneMetadata `json:"metadata"`
}

type googlePeopleResponse struct {
	PhoneNumbers []googlePeoplePhone `json:"phoneNumbers"`
}

func NewGoogleProvider(clientID, clientSecret string, scopes []string) *OAuth2Provider {
	// Default scopes if not configured.
	// user.phonenumbers.read lets us call People API to know whether the
	// Google account itself has a verified phone number on file.
	if len(scopes) == 0 {
		scopes = []string{
			"openid",
			"profile",
			"email",
			"https://www.googleapis.com/auth/user.phonenumbers.read",
		}
	}
	return &OAuth2Provider{
		name: domain.ProviderGoogle,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       scopes,
		},
		userURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		authOptions: []oauth2.AuthCodeOption{
			oauth2.AccessTypeOffline,
		},
		fetchUser: fetchGoogleUser,
	}
}

func fetchGoogleUser(ctx context.Context, client *http.Client, _ *oauth2.Token) (*port.ProviderUserInfo, error) {
	body, err := doGet(ctx, client, "https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetch google userinfo: %w", err)
	}
	var u googleUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("decode google user: %w", err)
	}
	var raw map[string]any
	_ = json.Unmarshal(body, &raw)
	if raw == nil {
		raw = map[string]any{}
	}
	emailVerified := u.EmailVerified || u.VerifiedEmail
	raw["email_verified"] = emailVerified

	// Best-effort fetch of phone numbers from People API. Failure to read
	// (e.g., user revoked scope) leaves the fields unset rather than failing
	// the whole login.
	if peopleBody, err := doGet(ctx, client, "https://people.googleapis.com/v1/people/me?personFields=phoneNumbers"); err == nil {
		var pr googlePeopleResponse
		if json.Unmarshal(peopleBody, &pr) == nil {
			hasPhone := false
			phoneVerified := false
			primaryPhone := ""
			for _, p := range pr.PhoneNumbers {
				hasPhone = true
				if p.Metadata.Verified {
					phoneVerified = true
				}
				if p.Metadata.Primary && primaryPhone == "" {
					primaryPhone = p.CanonicalForm
					if primaryPhone == "" {
						primaryPhone = p.Value
					}
				}
			}
			if primaryPhone == "" && len(pr.PhoneNumbers) > 0 {
				primaryPhone = pr.PhoneNumbers[0].CanonicalForm
				if primaryPhone == "" {
					primaryPhone = pr.PhoneNumbers[0].Value
				}
			}
			raw["has_phone"] = hasPhone
			raw["phone_verified"] = phoneVerified
			if primaryPhone != "" {
				raw["phone_number"] = primaryPhone
			}
		}
	}

	raw = normalizeRawProfile(raw, u.Email)
	return &port.ProviderUserInfo{
		ProviderUID:   u.ID,
		Email:         u.Email,
		EmailVerified: emailVerified,
		DisplayName:   u.Name,
		AvatarURL:     u.Picture,
		RawProfile:    raw,
	}, nil
}
