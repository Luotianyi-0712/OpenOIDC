package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/service"
)

func extractSessionToken(r *http.Request, cookieName string) string {
	if cookieName == "" {
		cookieName = "oidc_session"
	}
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func clearSessionCookie(w http.ResponseWriter, cookieName string) {
	if cookieName == "" {
		cookieName = "oidc_session"
	}
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func SessionAuth(sessionSvc *service.SessionService, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractSessionToken(r, cookieName)
			if token == "" {
				writeAuthError(w, "unauthenticated", "no session")
				return
			}
			sess, err := sessionSvc.ValidateSession(r.Context(), token)
			if err != nil {
				clearSessionCookie(w, cookieName)
				writeAuthError(w, "unauthenticated", "invalid session")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, sess.UserID)
			ctx = context.WithValue(ctx, SessionKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalSessionAuth(sessionSvc *service.SessionService, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractSessionToken(r, cookieName)
			if token != "" {
				if sess, err := sessionSvc.ValidateSession(r.Context(), token); err == nil {
					ctx := context.WithValue(r.Context(), UserIDKey, sess.UserID)
					ctx = context.WithValue(ctx, SessionKey, sess)
					r = r.WithContext(ctx)
				} else {
					clearSessionCookie(w, cookieName)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

var ErrNotAuthenticated = errors.New("not authenticated")

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	v := ctx.Value(UserIDKey)
	if v == nil {
		return uuid.Nil, ErrNotAuthenticated
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrNotAuthenticated
	}
	return id, nil
}

func GetSession(ctx context.Context) (*domain.UserSession, error) {
	v := ctx.Value(SessionKey)
	if v == nil {
		return nil, ErrNotAuthenticated
	}
	s, ok := v.(*domain.UserSession)
	if !ok {
		return nil, ErrNotAuthenticated
	}
	return s, nil
}

func writeAuthError(w http.ResponseWriter, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + msg + `"}}`))
}
