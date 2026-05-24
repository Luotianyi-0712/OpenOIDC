package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/service"
)

type AuthHandler struct {
	authSvc    *service.AuthService
	sessionSvc *service.SessionService
	sessionCfg config.SessionConfig
}

func NewAuthHandler(authSvc *service.AuthService, sessionSvc *service.SessionService, sessionCfg config.SessionConfig) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, sessionSvc: sessionSvc, sessionCfg: sessionCfg}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, err := h.authSvc.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		mapAuthError(w, err)
		return
	}
	JSON(w, http.StatusCreated, map[string]any{
		"id":             user.ID,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"email_verified": user.EmailVerified,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	h.revokeCurrentSession(r)
	ip := clientIP(r)
	ua := r.UserAgent()
	sess, err := h.authSvc.Login(r.Context(), req.Email, req.Password, ip, ua)
	if err != nil {
		mapAuthError(w, err)
		return
	}
	h.setSessionCookie(w, sess.SessionToken, sess.ExpiresAt)
	JSON(w, http.StatusOK, map[string]any{
		"session_token": sess.SessionToken,
		"expires_at":    sess.ExpiresAt,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractSessionTokenFromRequest(r, h.sessionCfg.CookieName)
	if token != "" {
		_ = h.authSvc.Logout(r.Context(), token)
	}
	h.clearSessionCookie(w)
	JSON(w, http.StatusOK, map[string]any{"logged_out": true})
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.authSvc.VerifyEmail(r.Context(), req.Token); err != nil {
		mapAuthError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	_ = h.authSvc.ForgotPassword(r.Context(), req.Email)
	JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.authSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		mapAuthError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"reset": true})
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	name := h.sessionCfg.CookieName
	if name == "" {
		name = "oidc_session"
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		Domain:   h.sessionCfg.CookieDomain,
		Expires:  expires,
		Secure:   h.sessionCfg.CookieSecure,
		HttpOnly: h.sessionCfg.CookieHTTPOnly,
		SameSite: sameSiteFromString(h.sessionCfg.CookieSameSite),
	}
	http.SetCookie(w, cookie)
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	name := h.sessionCfg.CookieName
	if name == "" {
		name = "oidc_session"
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   h.sessionCfg.CookieDomain,
		MaxAge:   -1,
		Secure:   h.sessionCfg.CookieSecure,
		HttpOnly: h.sessionCfg.CookieHTTPOnly,
		SameSite: sameSiteFromString(h.sessionCfg.CookieSameSite),
	}
	http.SetCookie(w, cookie)
}

func sameSiteFromString(s string) http.SameSite {
	switch s {
	case "strict", "Strict":
		return http.SameSiteStrictMode
	case "lax", "Lax":
		return http.SameSiteLaxMode
	case "none", "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (h *AuthHandler) revokeCurrentSession(r *http.Request) {
	token := extractSessionTokenFromRequest(r, h.sessionCfg.CookieName)
	if token != "" {
		_ = h.authSvc.Logout(r.Context(), token)
	}
}

func mapAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAlreadyExists):
		Error(w, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, service.ErrNotFound):
		Error(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, service.ErrAccountLockedOut):
		Error(w, http.StatusTooManyRequests, "account_locked", err.Error())
	case errors.Is(err, service.ErrAccountSuspended):
		Error(w, http.StatusForbidden, "account_suspended", "account is suspended")
	case errors.Is(err, service.ErrAccountDeleted):
		Error(w, http.StatusForbidden, "account_deleted", "account is deleted")
	case errors.Is(err, service.ErrEmailNotVerified):
		Error(w, http.StatusForbidden, "email_not_verified", "email not verified")
	case errors.Is(err, service.ErrInvalidToken):
		Error(w, http.StatusBadRequest, "invalid_token", "invalid or expired token")
	case errors.Is(err, service.ErrPasswordTooWeak):
		Error(w, http.StatusBadRequest, "password_too_weak", err.Error())
	case errors.Is(err, service.ErrInvalidEmail):
		Error(w, http.StatusBadRequest, "invalid_email", err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		Error(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal", "internal server error")
	}
}
