package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/config"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/service"
)

type PasskeyHandler struct {
	passkeySvc *service.PasskeyService
	sessionCfg config.SessionConfig
}

func NewPasskeyHandler(passkeySvc *service.PasskeyService, sessionCfg config.SessionConfig) *PasskeyHandler {
	return &PasskeyHandler{passkeySvc: passkeySvc, sessionCfg: sessionCfg}
}

type passkeyFinishRequest struct {
	SessionID string `json:"session_id"`
}

type passkeyRenameRequest struct {
	Name string `json:"name"`
}

// BeginRegister starts passkey registration (requires auth).
func (h *PasskeyHandler) BeginRegister(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}

	creation, sessionID, err := h.passkeySvc.BeginRegistration(r.Context(), userID)
	if err != nil {
		mapPasskeyError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"options":    creation,
		"session_id": sessionID,
	})
}

// FinishRegister completes passkey registration (requires auth).
func (h *PasskeyHandler) FinishRegister(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}

	sessionID := r.Header.Get("X-Passkey-Session")
	if sessionID == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "missing session id")
		return
	}

	cred, err := h.passkeySvc.FinishRegistration(r.Context(), userID, sessionID, r)
	if err != nil {
		mapPasskeyError(w, err)
		return
	}

	JSON(w, http.StatusCreated, map[string]any{
		"id":         cred.ID,
		"name":       cred.Name,
		"created_at": cred.CreatedAt,
	})
}

// BeginLogin starts passkey login (public, no auth required).
func (h *PasskeyHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	assertion, sessionID, err := h.passkeySvc.BeginLogin(r.Context())
	if err != nil {
		mapPasskeyError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"options":    assertion,
		"session_id": sessionID,
	})
}

// FinishLogin completes passkey login (public).
func (h *PasskeyHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Passkey-Session")
	if sessionID == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "missing session id")
		return
	}

	ip := clientIP(r)
	ua := r.UserAgent()

	sess, err := h.passkeySvc.FinishLogin(r.Context(), sessionID, ip, ua, r)
	if err != nil {
		mapPasskeyError(w, err)
		return
	}

	h.setSessionCookie(w, sess.SessionToken, sess.ExpiresAt)
	JSON(w, http.StatusOK, map[string]any{
		"session_token": sess.SessionToken,
		"expires_at":    sess.ExpiresAt,
	})
}

// ListPasskeys returns all passkeys for the authenticated user.
func (h *PasskeyHandler) ListPasskeys(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}

	creds, err := h.passkeySvc.ListCredentials(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	type passkeyItem struct {
		ID         uuid.UUID  `json:"id"`
		Name       string     `json:"name"`
		LastUsedAt *time.Time `json:"last_used_at,omitempty"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	items := make([]passkeyItem, 0, len(creds))
	for _, c := range creds {
		items = append(items, passkeyItem{
			ID:         c.ID,
			Name:       c.Name,
			LastUsedAt: c.LastUsedAt,
			CreatedAt:  c.CreatedAt,
		})
	}
	JSON(w, http.StatusOK, items)
}

// DeletePasskey removes a passkey credential.
func (h *PasskeyHandler) DeletePasskey(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}

	credID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", "invalid passkey id")
		return
	}

	if err := h.passkeySvc.DeleteCredential(r.Context(), userID, credID); err != nil {
		mapPasskeyError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// RenamePasskey renames a passkey credential.
func (h *PasskeyHandler) RenamePasskey(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}

	credID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", "invalid passkey id")
		return
	}

	var req passkeyRenameRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}

	if err := h.passkeySvc.RenameCredential(r.Context(), userID, credID, req.Name); err != nil {
		mapPasskeyError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"renamed": true})
}

func (h *PasskeyHandler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
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

func mapPasskeyError(w http.ResponseWriter, err error) {
	switch {
	case err == service.ErrPasskeyDisabled:
		Error(w, http.StatusForbidden, "passkey_disabled", "passkey is disabled")
	case err == service.ErrInvalidToken:
		Error(w, http.StatusBadRequest, "invalid_session", "challenge expired or invalid")
	case err == service.ErrAccountSuspended:
		Error(w, http.StatusForbidden, "account_suspended", "account is suspended")
	case err == service.ErrAccountDeleted:
		Error(w, http.StatusForbidden, "account_deleted", "account is deleted")
	case err == service.ErrNotFound:
		Error(w, http.StatusNotFound, "not_found", "passkey not found")
	default:
		Error(w, http.StatusInternalServerError, "internal", err.Error())
	}
}
