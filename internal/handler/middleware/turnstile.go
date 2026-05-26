package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/port"
)

const (
	turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	hcaptchaVerifyURL  = "https://hcaptcha.com/siteverify"
)

func Turnstile(settingsRepo port.SettingsRepository) func(http.Handler) http.Handler {
	return Captcha(settingsRepo)
}

func Captcha(settingsRepo port.SettingsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if settingsRepo == nil {
				next.ServeHTTP(w, r)
				return
			}

			provider := settingValue(r, settingsRepo, "captcha_provider")
			if provider == "" {
				provider = "turnstile"
			}
			if enabled := settingValue(r, settingsRepo, "captcha_enabled"); enabled == "false" {
				next.ServeHTTP(w, r)
				return
			}

			secret := settingValue(r, settingsRepo, "captcha_secret_key")
			if secret == "" && provider == "turnstile" {
				secret = settingValue(r, settingsRepo, "turnstile_secret_key")
			}
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}

			token := r.Header.Get("X-Captcha-Token")
			if token == "" {
				token = r.Header.Get("X-Turnstile-Token")
			}
			if token == "" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"captcha_required","message":"human verification required"}}`))
				return
			}

			if !verifyCaptcha(r.Context(), provider, secret, token, clientIP(r)) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"captcha_failed","message":"human verification failed"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func settingValue(r *http.Request, settingsRepo port.SettingsRepository, key string) string {
	setting, err := settingsRepo.Get(r.Context(), key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(setting.Value)
}

func verifyCaptcha(ctx context.Context, provider, secret, token, remoteIP string) bool {
	verifyURL := turnstileVerifyURL
	if provider == "hcaptcha" {
		verifyURL = hcaptchaVerifyURL
	}

	data := url.Values{}
	data.Set("secret", secret)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(verifyURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}
	slog.Debug("captcha verify", "provider", provider, "success", result.Success)
	return result.Success
}
