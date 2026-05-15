package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

const (
	qqAuthURL     = "https://graph.qq.com/oauth2.0/authorize"
	qqTokenURL    = "https://graph.qq.com/oauth2.0/token"
	qqOpenIDURL   = "https://graph.qq.com/oauth2.0/me"
	qqUserInfoURL = "https://graph.qq.com/user/get_user_info"
)

type QQProvider struct {
	clientID     string
	clientSecret string
}

func NewQQProvider(clientID, clientSecret string) *QQProvider {
	return &QQProvider{clientID: clientID, clientSecret: clientSecret}
}

func (p *QQProvider) Name() string { return domain.ProviderQQ }

func (p *QQProvider) BeginAuth(_ context.Context, state, redirectURL string) (string, error) {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", redirectURL)
	params.Set("state", state)
	params.Set("scope", "get_user_info")
	return qqAuthURL + "?" + params.Encode(), nil
}

type qqUserInfo struct {
	Ret           int    `json:"ret"`
	Msg           string `json:"msg"`
	Nickname      string `json:"nickname"`
	FigureURLQQ2  string `json:"figureurl_qq_2"`
	FigureURLQQ1  string `json:"figureurl_qq_1"`
	FigureURLQQ   string `json:"figureurl_qq"`
}

func (p *QQProvider) CompleteAuth(ctx context.Context, r *http.Request) (*port.ProviderUserInfo, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	code := r.FormValue("code")
	if code == "" {
		return nil, fmt.Errorf("missing code")
	}
	redirectURL := redirectFromRequest(r)

	accessToken, _, err := p.exchangeCode(ctx, code, redirectURL)
	if err != nil {
		return nil, err
	}

	openID, err := p.fetchOpenID(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	info, err := p.fetchUserInfo(ctx, accessToken, openID)
	if err != nil {
		return nil, err
	}

	avatar := info.FigureURLQQ2
	if avatar == "" {
		avatar = info.FigureURLQQ1
	}
	if avatar == "" {
		avatar = info.FigureURLQQ
	}

	raw := map[string]any{
		"openid":   openID,
		"nickname": info.Nickname,
		"figureurl_qq_2": info.FigureURLQQ2,
	}

	return &port.ProviderUserInfo{
		ProviderUID: openID,
		DisplayName: info.Nickname,
		AvatarURL:   avatar,
		RawProfile:  raw,
	}, nil
}

func (p *QQProvider) SupportsRefresh() bool { return true }

func (p *QQProvider) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	if refreshToken == "" {
		return "", "", fmt.Errorf("empty refresh token")
	}
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("client_id", p.clientID)
	params.Set("client_secret", p.clientSecret)
	params.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qqTokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	return parseQQTokenResponse(string(body))
}

func (p *QQProvider) exchangeCode(ctx context.Context, code, redirectURL string) (string, string, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", p.clientID)
	params.Set("client_secret", p.clientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qqTokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	return parseQQTokenResponse(string(body))
}

// parseQQTokenResponse handles the query-string response, e.g.:
//   access_token=FE04...&expires_in=7776000&refresh_token=88E4...
// On error the body may also be JSON: {"error":1,"error_description":"..."}
func parseQQTokenResponse(body string) (string, string, error) {
	body = strings.TrimSpace(body)
	if strings.HasPrefix(body, "{") {
		var errResp struct {
			Error            json.RawMessage `json:"error"`
			ErrorDescription string          `json:"error_description"`
		}
		if err := json.Unmarshal([]byte(body), &errResp); err == nil && len(errResp.Error) > 0 {
			return "", "", fmt.Errorf("qq token error: %s: %s", string(errResp.Error), errResp.ErrorDescription)
		}
	}
	values, err := url.ParseQuery(body)
	if err != nil {
		return "", "", fmt.Errorf("parse qq token response: %w", err)
	}
	at := values.Get("access_token")
	if at == "" {
		return "", "", fmt.Errorf("qq response missing access_token: %s", body)
	}
	return at, values.Get("refresh_token"), nil
}

func (p *QQProvider) fetchOpenID(ctx context.Context, accessToken string) (string, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qqOpenIDURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch openid: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Response is JSONP: callback({"client_id":"...","openid":"..."});
	s := string(body)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("unexpected qq openid response: %s", s)
	}
	var data struct {
		Error            int    `json:"error"`
		ErrorDescription string `json:"error_description"`
		OpenID           string `json:"openid"`
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &data); err != nil {
		return "", fmt.Errorf("decode qq openid: %w", err)
	}
	if data.Error != 0 {
		return "", fmt.Errorf("qq openid error %d: %s", data.Error, data.ErrorDescription)
	}
	if data.OpenID == "" {
		return "", fmt.Errorf("qq openid response missing openid")
	}
	return data.OpenID, nil
}

func (p *QQProvider) fetchUserInfo(ctx context.Context, accessToken, openID string) (*qqUserInfo, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("oauth_consumer_key", p.clientID)
	params.Set("openid", openID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qqUserInfoURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch qq user: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info qqUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decode qq user: %w", err)
	}
	if info.Ret != 0 {
		return nil, fmt.Errorf("qq user_info error %d: %s", info.Ret, info.Msg)
	}
	return &info, nil
}

var _ port.SocialProvider = (*QQProvider)(nil)
