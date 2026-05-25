package middleware

import (
	"net/http"
	"strconv"

	"github.com/anthropic/oidc-platform/internal/port"
)

func AdminOnly(userRepo port.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := GetUserID(r.Context())
			if err != nil {
				writeAuthError(w, "unauthenticated", "no session")
				return
			}
			user, err := userRepo.GetByID(r.Context(), userID)
			if err != nil || user == nil {
				writeAuthError(w, "unauthenticated", "user not found")
				return
			}
			if !user.IsAdmin() {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"forbidden","message":"admin only"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func DeveloperOnly(userRepo port.UserRepository, settingsRepo port.SettingsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := GetUserID(r.Context())
			if err != nil {
				writeAuthError(w, "unauthenticated", "no session")
				return
			}
			user, err := userRepo.GetByID(r.Context(), userID)
			if err != nil || user == nil {
				writeAuthError(w, "unauthenticated", "user not found")
				return
			}
			if user.IsAdmin() {
				next.ServeHTTP(w, r)
				return
			}
			if !user.EmailVerified {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"email_not_verified","message":"developer access requires a verified email"}}`))
				return
			}
			minLevel := developerMinTrustLevel(r, settingsRepo)
			if user.SecurityLevel < minLevel {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"developer_level_required","message":"developer access requires higher trust level"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func developerMinTrustLevel(r *http.Request, settingsRepo port.SettingsRepository) int {
	if settingsRepo == nil {
		return 1
	}
	setting, err := settingsRepo.Get(r.Context(), "developer_min_trust_level")
	if err != nil || setting == nil || setting.Value == "" {
		return 1
	}
	level, err := strconv.Atoi(setting.Value)
	if err != nil || level < 0 {
		return 1
	}
	return level
}
