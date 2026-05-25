package social

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/port"
)

// UserInfoFetcher allows providers to override how user information is fetched
// after obtaining the OAuth2 token (e.g., to call multiple endpoints).
type UserInfoFetcher func(ctx context.Context, client *http.Client, token *oauth2.Token) (*port.ProviderUserInfo, error)

// OAuth2Provider is a generic OAuth2 social provider implementation.
type OAuth2Provider struct {
	name      string
	config    *oauth2.Config
	userURL   string
	parseUser func(body []byte) (*port.ProviderUserInfo, error)
	// fetchUser, when set, takes precedence over userURL+parseUser.
	fetchUser UserInfoFetcher
	// authOptions are additional query parameters appended to the auth URL
	// (e.g., access_type=offline for Google).
	authOptions []oauth2.AuthCodeOption
}

func (p *OAuth2Provider) Name() string {
	return p.name
}

func (p *OAuth2Provider) BeginAuth(ctx context.Context, state string, redirectURL string) (string, error) {
	cfg := p.cloneWithRedirect(redirectURL)
	return cfg.AuthCodeURL(state, p.authOptions...), nil
}

func (p *OAuth2Provider) CompleteAuth(ctx context.Context, r *http.Request) (*port.ProviderUserInfo, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	if errParam := r.FormValue("error"); errParam != "" {
		desc := r.FormValue("error_description")
		return nil, fmt.Errorf("oauth2 callback error: %s: %s", errParam, desc)
	}
	code := r.FormValue("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}
	redirectURL := redirectFromRequest(r)
	cfg := p.cloneWithRedirect(redirectURL)

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	client := cfg.Client(ctx, token)

	var info *port.ProviderUserInfo
	if p.fetchUser != nil {
		info, err = p.fetchUser(ctx, client, token)
	} else {
		body, fetchErr := doGet(ctx, client, p.userURL)
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch user info: %w", fetchErr)
		}
		info, err = p.parseUser(body)
	}
	if err != nil {
		return nil, fmt.Errorf("parse user info: %w", err)
	}
	info.Token = oauthTokenInfo(token, p.config.Scopes)
	return info, nil
}

func (p *OAuth2Provider) SupportsRefresh() bool {
	return true
}

func (p *OAuth2Provider) RefreshToken(ctx context.Context, refreshToken string) (*port.ProviderTokenInfo, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("empty refresh token")
	}
	tokenSource := p.config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	return oauthTokenInfo(newToken, p.config.Scopes), nil
}

func (p *OAuth2Provider) ValidateToken(ctx context.Context, accessToken string) (*port.ProviderUserInfo, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("empty access token")
	}
	token := &oauth2.Token{AccessToken: accessToken, TokenType: "Bearer"}
	client := p.config.Client(ctx, token)
	if p.fetchUser != nil {
		return p.fetchUser(ctx, client, token)
	}
	body, err := doGet(ctx, client, p.userURL)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	info, err := p.parseUser(body)
	if err != nil {
		return nil, fmt.Errorf("parse user info: %w", err)
	}
	return info, nil
}

func oauthTokenInfo(token *oauth2.Token, scopes []string) *port.ProviderTokenInfo {
	if token == nil {
		return nil
	}
	var expiry *time.Time
	if !token.Expiry.IsZero() {
		exp := token.Expiry.UTC()
		expiry = &exp
	}
	return &port.ProviderTokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       expiry,
		TokenType:    token.TokenType,
		Scopes:       append([]string(nil), scopes...),
	}
}

func (p *OAuth2Provider) cloneWithRedirect(redirectURL string) *oauth2.Config {
	cfg := *p.config
	if redirectURL != "" {
		cfg.RedirectURL = redirectURL
	}
	return &cfg
}

func doGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// redirectFromRequest reconstructs the original redirect_uri from the callback
// request so that token exchange uses the same value the user was sent to.
func redirectFromRequest(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	scheme := "https"
	if r.TLS == nil {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	u := *r.URL
	u.RawQuery = ""
	u.Fragment = ""
	u.Scheme = scheme
	u.Host = host
	return u.String()
}

var _ port.SocialProvider = (*OAuth2Provider)(nil)
