package middleware

import (
	"net/http"

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
