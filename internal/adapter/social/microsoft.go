package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
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

func NewMicrosoftProvider(clientID, clientSecret, tenantID string, scopes []string) *OAuth2Provider {
	if tenantID == "" {
		tenantID = "common"
	}
	// Default scopes if not configured
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email", "User.Read", "offline_access"}
	}
	configuredTenant := tenantID
	return &OAuth2Provider{
		name: domain.ProviderMicrosoft,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/" + tenantID + "/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/" + tenantID + "/oauth2/v2.0/token",
			},
			Scopes: scopes,
		},
		userURL: "https://graph.microsoft.com/v1.0/me",
		fetchUser: func(ctx context.Context, client *http.Client, token *oauth2.Token) (*port.ProviderUserInfo, error) {
			body, err := doGet(ctx, client, "https://graph.microsoft.com/v1.0/me")
			if err != nil {
				return nil, fmt.Errorf("fetch microsoft user: %w", err)
			}
			return parseMicrosoftUser(body, token, configuredTenant)
		},
	}
}

func parseMicrosoftUser(body []byte, token *oauth2.Token, configuredTenant string) (*port.ProviderUserInfo, error) {
	var u microsoftUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("decode microsoft user: %w", err)
	}

	raw := map[string]any{}
	_ = json.Unmarshal(body, &raw)
	if raw == nil {
		raw = map[string]any{}
	}

	if claims, source := microsoftClaimsFromToken(token); len(claims) > 0 {
		raw["microsoft_claim_source"] = source
		raw[source+"_claims"] = claims
		mergeMicrosoftClaims(raw, claims)
	}
	if configuredTenant != "" {
		raw["configured_tenant"] = configuredTenant
	}

	email := microsoftFirstNonEmpty(u.Mail, u.UserPrincipalName, microsoftClaimString(raw, "email"), microsoftClaimString(raw, "preferred_username"), microsoftClaimString(raw, "upn"))
	raw = normalizeRawProfile(raw, email)

	uid := microsoftFirstNonEmpty(u.ID, microsoftClaimString(raw, "oid"), microsoftClaimString(raw, "sub"))
	if uid == "" {
		return nil, fmt.Errorf("microsoft user missing id")
	}
	display := microsoftFirstNonEmpty(u.DisplayName, microsoftClaimString(raw, "name"), email)
	emailVerified, _ := rawProfileBool(raw, "email_verified")

	return &port.ProviderUserInfo{
		ProviderUID:   uid,
		Email:         email,
		EmailVerified: emailVerified,
		DisplayName:   display,
		RawProfile:    raw,
	}, nil
}

func microsoftClaimsFromToken(token *oauth2.Token) (map[string]any, string) {
	if token == nil {
		return nil, ""
	}

	candidates := make([]struct {
		source string
		value  string
	}, 0, 2)
	if idToken, ok := token.Extra("id_token").(string); ok {
		candidates = append(candidates, struct {
			source string
			value  string
		}{source: "id_token", value: idToken})
	}
	candidates = append(candidates, struct {
		source string
		value  string
	}{source: "access_token", value: token.AccessToken})

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	for _, candidate := range candidates {
		value := strings.TrimSpace(candidate.value)
		if value == "" {
			continue
		}
		claims := jwt.MapClaims{}
		if _, _, err := parser.ParseUnverified(value, claims); err != nil {
			continue
		}
		out := make(map[string]any, len(claims))
		for key, value := range claims {
			out[key] = value
		}
		return out, candidate.source
	}
	return nil, ""
}

func mergeMicrosoftClaims(raw map[string]any, claims map[string]any) {
	for _, key := range []string{"tid", "oid", "sub", "preferred_username", "email", "upn", "name"} {
		if value := microsoftClaimString(claims, key); value != "" {
			raw[key] = value
		}
	}
	if tenantID := microsoftClaimString(claims, "tid"); tenantID != "" {
		raw["tenant"] = tenantID
		raw["tenant_id"] = tenantID
	}
	if emailVerified, ok := microsoftClaimBool(claims, "email_verified"); ok {
		raw["email_verified"] = emailVerified
	}
}

func microsoftClaimString(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	value, ok := claims[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func microsoftClaimBool(claims map[string]any, key string) (bool, bool) {
	if claims == nil {
		return false, false
	}
	value, ok := claims[key]
	if !ok || value == nil {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed := strings.ToLower(strings.TrimSpace(v))
		if parsed == "true" || parsed == "1" {
			return true, true
		}
		if parsed == "false" || parsed == "0" {
			return false, true
		}
	}
	return false, false
}

func microsoftFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
