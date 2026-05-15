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

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Turnstile verifies Cloudflare Turnstile tokens.
// If turnstile_secret_key is not configured in settings, the check is skipped.
func Turnstile(settingsRepo port.SettingsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if settingsRepo == nil {
				next.ServeHTTP(w, r)
				return
			}

			secret, err := settingsRepo.Get(r.Context(), "turnstile_secret_key")
			if err != nil || secret.Value == "" {
				// Turnstile not configured, skip verification.
				next.ServeHTTP(w, r)
				return
			}

			token := r.Header.Get("X-Turnstile-Token")
			if token == "" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"captcha_required","message":"human verification required"}}`))
				return
			}

			if !verifyTurnstile(r.Context(), secret.Value, token, clientIP(r)) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"captcha_failed","message":"human verification failed"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func verifyTurnstile(ctx context.Context, secret, token, remoteIP string) bool {
	data := url.Values{}
	data.Set("secret", secret)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(turnstileVerifyURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
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
	slog.Debug("turnstile verify", "success", result.Success)
	return result.Success
}
