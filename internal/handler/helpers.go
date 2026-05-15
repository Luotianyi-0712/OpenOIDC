package handler

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/anthropic/oidc-platform/internal/service"
)

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return strings.TrimSpace(rip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func mapSocialError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProviderDisabled):
		Error(w, http.StatusBadRequest, "provider_disabled", err.Error())
	case errors.Is(err, service.ErrAlreadyBound):
		Error(w, http.StatusConflict, "already_bound", err.Error())
	case errors.Is(err, service.ErrBindingNotFound):
		Error(w, http.StatusNotFound, "binding_not_found", err.Error())
	case errors.Is(err, service.ErrAccountSuspended):
		Error(w, http.StatusForbidden, "account_suspended", err.Error())
	case errors.Is(err, service.ErrAccountDeleted):
		Error(w, http.StatusForbidden, "account_deleted", err.Error())
	case errors.Is(err, service.ErrInvalidToken):
		Error(w, http.StatusBadRequest, "invalid_state", err.Error())
	case errors.Is(err, service.ErrNotFound):
		Error(w, http.StatusNotFound, "not_found", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal", err.Error())
	}
}

func extractSessionTokenFromRequest(r *http.Request, cookieName string) string {
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
