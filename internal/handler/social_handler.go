package handler

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type SocialHandler struct {
	socialSvc      *service.SocialService
	socialRegistry port.SocialProviderRegistry
	sessionSvc     *service.SessionService
	sessionCfg     config.SessionConfig
}

func NewSocialHandler(socialSvc *service.SocialService, socialRegistry port.SocialProviderRegistry, sessionSvc *service.SessionService, sessionCfg config.SessionConfig) *SocialHandler {
	return &SocialHandler{socialSvc: socialSvc, socialRegistry: socialRegistry, sessionSvc: sessionSvc, sessionCfg: sessionCfg}
}

// ListEnabled handles GET /api/v1/social/providers (public, no auth).
// Returns only enabled providers with their display info.
func (h *SocialHandler) ListEnabled(w http.ResponseWriter, r *http.Request) {
	names := h.socialRegistry.List()

	displayNames := map[string]string{
		domain.ProviderGitHub:    "GitHub",
		domain.ProviderGoogle:    "Google",
		domain.ProviderGitLab:    "GitLab",
		domain.ProviderGitee:     "Gitee",
		domain.ProviderDiscord:   "Discord",
		domain.ProviderTelegram:  "Telegram",
		domain.ProviderMicrosoft: "Microsoft",
		domain.ProviderApple:     "Apple",
		domain.ProviderQQ:        "QQ",
		domain.ProviderWeChat:    "WeChat",
		domain.ProviderPhone:     "Phone",
	}

	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		dn := displayNames[name]
		if dn == "" {
			dn = name
		}
		out = append(out, map[string]any{
			"name":         name,
			"display_name": dn,
		})
	}
	JSON(w, http.StatusOK, out)
}

// Begin handles GET /api/v1/social/{provider}/begin
// Redirects the browser to the OAuth provider's authorization page.
func (h *SocialHandler) Begin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "missing provider")
		return
	}

	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}

	if userID, err := mw.GetUserID(r.Context()); err == nil {
		authURL, err := h.socialSvc.BeginBinding(r.Context(), userID, provider, returnTo)
		if err != nil {
			h.redirectError(w, r, returnTo, err)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
		return
	}

	authURL, err := h.socialSvc.BeginSocialLogin(r.Context(), provider, returnTo)
	if err != nil {
		h.redirectError(w, r, returnTo, err)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles GET /api/v1/social/{provider}/callback
// This is where the OAuth provider redirects back after user authorization.
func (h *SocialHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "missing provider")
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		h.redirectError(w, r, "/me/bindings", service.ErrInvalidToken)
		return
	}

	stateInfo, err := h.socialSvc.PeekState(r.Context(), state)
	if err != nil {
		h.redirectError(w, r, "/me/bindings", err)
		return
	}

	returnTo := stateInfo.ReturnTo
	if returnTo == "" {
		returnTo = "/"
	}

	switch stateInfo.Mode {
	case "bind":
		_, err := h.socialSvc.CompleteBinding(r.Context(), stateInfo.UserID, provider, r)
		if err != nil {
			h.redirectError(w, r, returnTo, err)
			return
		}
		cookieName := h.sessionCfg.CookieName
		if cookieName == "" {
			cookieName = "oidc_session"
		}
		if token := extractSessionTokenFromRequest(r, cookieName); token != "" {
			if sess, err := h.sessionSvc.ValidateSession(r.Context(), token); err == nil {
				h.setSessionCookie(w, sess.SessionToken, sess.ExpiresAt)
			}
		}
		h.redirectSuccess(w, r, returnTo, "bind_success")

	case "login":
		if token := extractSessionTokenFromRequest(r, h.sessionCfg.CookieName); token != "" {
			_ = h.sessionSvc.RevokeByToken(r.Context(), token)
		}
		ip := clientIP(r)
		ua := r.UserAgent()
		sess, _, err := h.socialSvc.CompleteSocialLogin(r.Context(), provider, r, ip, ua)
		if err != nil {
			h.redirectError(w, r, "/login", err)
			return
		}
		h.setSessionCookie(w, sess.SessionToken, sess.ExpiresAt)
		h.redirectSuccess(w, r, returnTo, "login_success")

	default:
		h.redirectError(w, r, "/", service.ErrInvalidToken)
	}
}

func (h *SocialHandler) redirectSuccess(w http.ResponseWriter, r *http.Request, returnTo, result string) {
	u, err := url.Parse(returnTo)
	if err != nil || !isRelativePath(returnTo) {
		u = &url.URL{Path: "/"}
	}
	q := u.Query()
	q.Set("result", result)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *SocialHandler) redirectError(w http.ResponseWriter, r *http.Request, returnTo string, err error) {
	u, parseErr := url.Parse(returnTo)
	if parseErr != nil || !isRelativePath(returnTo) {
		u = &url.URL{Path: "/"}
	}
	q := u.Query()
	q.Set("error", errorCode(err))
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func isRelativePath(s string) bool {
	if s == "" {
		return false
	}
	return s[0] == '/'
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrProviderDisabled):
		return "provider_disabled"
	case errors.Is(err, service.ErrAlreadyBound):
		return "already_bound"
	case errors.Is(err, service.ErrBindingNotFound):
		return "binding_not_found"
	case errors.Is(err, service.ErrAccountSuspended):
		return "account_suspended"
	case errors.Is(err, service.ErrAccountDeleted):
		return "account_deleted"
	case errors.Is(err, service.ErrRegistrationDisabled):
		return "registration_disabled"
	case errors.Is(err, service.ErrInvalidToken):
		return "invalid_state"
	default:
		return "internal_error"
	}
}

func (h *SocialHandler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	name := h.sessionCfg.CookieName
	if name == "" {
		name = "oidc_session"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		Domain:   h.sessionCfg.CookieDomain,
		Expires:  expires,
		Secure:   h.sessionCfg.CookieSecure,
		HttpOnly: h.sessionCfg.CookieHTTPOnly,
		SameSite: sameSiteFromString(h.sessionCfg.CookieSameSite),
	})
}
