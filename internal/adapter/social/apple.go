package social

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

const (
	appleAuthURL  = "https://appleid.apple.com/auth/authorize"
	appleTokenURL = "https://appleid.apple.com/auth/token"
	appleAudience = "https://appleid.apple.com"
)

type AppleProvider struct {
	clientID   string
	teamID     string
	keyID      string
	privateKey *ecdsa.PrivateKey
}

func NewAppleProvider(clientID, teamID, keyID string, privateKey *ecdsa.PrivateKey) *AppleProvider {
	return &AppleProvider{
		clientID:   clientID,
		teamID:     teamID,
		keyID:      keyID,
		privateKey: privateKey,
	}
}

func (p *AppleProvider) Name() string { return domain.ProviderApple }

func (p *AppleProvider) BeginAuth(_ context.Context, state, redirectURL string) (string, error) {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "name email")
	params.Set("response_mode", "form_post")
	params.Set("state", state)
	return appleAuthURL + "?" + params.Encode(), nil
}

type appleTokenResp struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type appleIDTokenClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	jwt.RegisteredClaims
}

type appleUserPayload struct {
	Name struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
	Email string `json:"email"`
}

func (p *AppleProvider) CompleteAuth(ctx context.Context, r *http.Request) (*port.ProviderUserInfo, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	if errParam := r.FormValue("error"); errParam != "" {
		return nil, fmt.Errorf("apple oauth error: %s", errParam)
	}
	code := r.FormValue("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	secret, err := p.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("generate apple client secret: %w", err)
	}

	redirectURL := redirectFromRequest(r)

	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", secret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	if redirectURL != "" {
		form.Set("redirect_uri", redirectURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokenResp appleTokenResp
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode apple token response: %w", err)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("apple token error: %s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.IDToken == "" {
		return nil, fmt.Errorf("apple response missing id_token")
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var claims appleIDTokenClaims
	if _, _, err := parser.ParseUnverified(tokenResp.IDToken, &claims); err != nil {
		return nil, fmt.Errorf("parse id_token: %w", err)
	}
	if claims.Sub == "" {
		return nil, fmt.Errorf("apple id_token missing sub")
	}

	displayName := ""
	email := claims.Email
	if userParam := r.FormValue("user"); userParam != "" {
		var up appleUserPayload
		if err := json.Unmarshal([]byte(userParam), &up); err == nil {
			displayName = strings.TrimSpace(up.Name.FirstName + " " + up.Name.LastName)
			if up.Email != "" {
				email = up.Email
			}
		}
	}

	raw := map[string]any{
		"sub":           claims.Sub,
		"email":         claims.Email,
		"id_token":      tokenResp.IDToken,
		"refresh_token": tokenResp.RefreshToken,
	}

	return &port.ProviderUserInfo{
		ProviderUID: claims.Sub,
		Email:       email,
		DisplayName: displayName,
		RawProfile:  raw,
	}, nil
}

func (p *AppleProvider) SupportsRefresh() bool { return true }

func (p *AppleProvider) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	if refreshToken == "" {
		return "", "", fmt.Errorf("empty refresh token")
	}
	secret, err := p.generateClientSecret()
	if err != nil {
		return "", "", err
	}
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", secret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	var tokenResp appleTokenResp
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", err
	}
	if tokenResp.Error != "" {
		return "", "", fmt.Errorf("apple refresh error: %s", tokenResp.Error)
	}
	return tokenResp.AccessToken, tokenResp.RefreshToken, nil
}

func (p *AppleProvider) generateClientSecret() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    p.teamID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		Audience:  jwt.ClaimStrings{appleAudience},
		Subject:   p.clientID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = p.keyID
	return token.SignedString(p.privateKey)
}

var _ port.SocialProvider = (*AppleProvider)(nil)
