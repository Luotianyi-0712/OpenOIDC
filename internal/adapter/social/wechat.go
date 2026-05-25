package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

const (
	wechatAuthURL  = "https://open.weixin.qq.com/connect/qrconnect"
	wechatTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wechatUserURL  = "https://api.weixin.qq.com/sns/userinfo"
	wechatRefresh  = "https://api.weixin.qq.com/sns/oauth2/refresh_token"
)

type WeChatProvider struct {
	appID     string
	appSecret string
}

func NewWeChatProvider(appID, appSecret string) *WeChatProvider {
	return &WeChatProvider{appID: appID, appSecret: appSecret}
}

func (p *WeChatProvider) Name() string { return domain.ProviderWeChat }

func (p *WeChatProvider) BeginAuth(_ context.Context, state, redirectURL string) (string, error) {
	params := url.Values{}
	params.Set("appid", p.appID)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "snsapi_login")
	params.Set("state", state)
	return wechatAuthURL + "?" + params.Encode() + "#wechat_redirect", nil
}

type wechatTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserInfo struct {
	OpenID     string   `json:"openid"`
	UnionID    string   `json:"unionid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	ErrCode    int      `json:"errcode"`
	ErrMsg     string   `json:"errmsg"`
}

func (p *WeChatProvider) CompleteAuth(ctx context.Context, r *http.Request) (*port.ProviderUserInfo, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	code := r.FormValue("code")
	if code == "" {
		return nil, fmt.Errorf("missing code")
	}

	params := url.Values{}
	params.Set("appid", p.appID)
	params.Set("secret", p.appSecret)
	params.Set("code", code)
	params.Set("grant_type", "authorization_code")

	tokenBody, err := wechatGet(ctx, wechatTokenURL+"?"+params.Encode())
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	var tok wechatTokenResp
	if err := json.Unmarshal(tokenBody, &tok); err != nil {
		return nil, fmt.Errorf("decode wechat token: %w", err)
	}
	if tok.ErrCode != 0 {
		return nil, fmt.Errorf("wechat token error %d: %s", tok.ErrCode, tok.ErrMsg)
	}

	userParams := url.Values{}
	userParams.Set("access_token", tok.AccessToken)
	userParams.Set("openid", tok.OpenID)
	userParams.Set("lang", "zh_CN")

	userBody, err := wechatGet(ctx, wechatUserURL+"?"+userParams.Encode())
	if err != nil {
		return nil, fmt.Errorf("fetch wechat user: %w", err)
	}
	var info wechatUserInfo
	if err := json.Unmarshal(userBody, &info); err != nil {
		return nil, fmt.Errorf("decode wechat user: %w", err)
	}
	if info.ErrCode != 0 {
		return nil, fmt.Errorf("wechat user error %d: %s", info.ErrCode, info.ErrMsg)
	}

	uid := info.UnionID
	if uid == "" {
		uid = info.OpenID
	}

	var raw map[string]any
	_ = json.Unmarshal(userBody, &raw)

	return &port.ProviderUserInfo{
		ProviderUID: uid,
		DisplayName: info.Nickname,
		AvatarURL:   info.HeadImgURL,
		RawProfile:  raw,
		Token:       wechatTokenInfo(tok),
	}, nil
}

func (p *WeChatProvider) SupportsRefresh() bool { return true }

func (p *WeChatProvider) RefreshToken(ctx context.Context, refreshToken string) (*port.ProviderTokenInfo, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("empty refresh token")
	}
	params := url.Values{}
	params.Set("appid", p.appID)
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)

	body, err := wechatGet(ctx, wechatRefresh+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var tok wechatTokenResp
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, err
	}
	if tok.ErrCode != 0 {
		return nil, fmt.Errorf("wechat refresh error %d: %s", tok.ErrCode, tok.ErrMsg)
	}
	return wechatTokenInfo(tok), nil
}

func (p *WeChatProvider) ValidateToken(ctx context.Context, accessToken string) (*port.ProviderUserInfo, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("empty access token")
	}
	return nil, fmt.Errorf("wechat validation requires openid from token response")
}

func wechatTokenInfo(tok wechatTokenResp) *port.ProviderTokenInfo {
	var expiry *time.Time
	if tok.ExpiresIn > 0 {
		exp := time.Now().UTC().Add(time.Duration(tok.ExpiresIn) * time.Second)
		expiry = &exp
	}
	var scopes []string
	if tok.Scope != "" {
		scopes = strings.FieldsFunc(tok.Scope, func(r rune) bool { return r == ',' || r == ' ' })
	}
	return &port.ProviderTokenInfo{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       expiry,
		TokenType:    "Bearer",
		Scopes:       scopes,
	}
}

func wechatGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

var _ port.SocialProvider = (*WeChatProvider)(nil)
