package social

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

const telegramAuthMaxAge = 24 * time.Hour

type TelegramProvider struct {
	botToken string
}

func NewTelegramProvider(botToken string) *TelegramProvider {
	return &TelegramProvider{botToken: botToken}
}

func (p *TelegramProvider) Name() string { return domain.ProviderTelegram }

// BeginAuth: Telegram uses a frontend Login Widget; there is no server-side
// redirect URL to build, so we return an empty string.
func (p *TelegramProvider) BeginAuth(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

func (p *TelegramProvider) CompleteAuth(_ context.Context, r *http.Request) (*port.ProviderUserInfo, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}

	values := map[string]string{}
	for key := range r.Form {
		values[key] = r.FormValue(key)
	}

	hashGiven, ok := values["hash"]
	if !ok || hashGiven == "" {
		return nil, fmt.Errorf("missing telegram hash")
	}
	delete(values, "hash")

	if !p.verifyHash(values, hashGiven) {
		return nil, fmt.Errorf("invalid telegram hash")
	}

	authDateStr := values["auth_date"]
	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid auth_date: %w", err)
	}
	if time.Since(time.Unix(authDate, 0)) > telegramAuthMaxAge {
		return nil, fmt.Errorf("telegram auth data expired")
	}

	id := values["id"]
	if id == "" {
		return nil, fmt.Errorf("missing telegram user id")
	}

	first := values["first_name"]
	last := values["last_name"]
	username := values["username"]
	photo := values["photo_url"]

	display := strings.TrimSpace(first + " " + last)
	if display == "" {
		display = username
	}

	raw := make(map[string]any, len(values))
	for k, v := range values {
		raw[k] = v
	}

	return &port.ProviderUserInfo{
		ProviderUID: id,
		DisplayName: display,
		AvatarURL:   photo,
		RawProfile:  raw,
	}, nil
}

func (p *TelegramProvider) SupportsRefresh() bool { return false }

func (p *TelegramProvider) RefreshToken(_ context.Context, _ string) (*port.ProviderTokenInfo, error) {
	return nil, fmt.Errorf("telegram does not support token refresh")
}

func (p *TelegramProvider) verifyHash(values map[string]string, hashGiven string) bool {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+values[k])
	}
	dataCheckString := strings.Join(parts, "\n")

	secret := sha256.Sum256([]byte(p.botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(hashGiven))
}

var _ port.SocialProvider = (*TelegramProvider)(nil)
