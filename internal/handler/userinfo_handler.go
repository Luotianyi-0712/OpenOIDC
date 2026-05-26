package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type UserInfoHandler struct {
	userRepo    port.UserRepository
	socialSvc   *service.SocialService
	securitySvc *service.SecurityLevelService
	accessCtrl  *service.AccessControlService
	authSvc     *service.AuthService
	sessionSvc  *service.SessionService
	consentRepo port.ConsentRepository
}

func NewUserInfoHandler(
	userRepo port.UserRepository,
	socialSvc *service.SocialService,
	securitySvc *service.SecurityLevelService,
	accessCtrl *service.AccessControlService,
	authSvc *service.AuthService,
	sessionSvc *service.SessionService,
	consentRepo port.ConsentRepository,
) *UserInfoHandler {
	return &UserInfoHandler{
		userRepo:    userRepo,
		socialSvc:   socialSvc,
		securitySvc: securitySvc,
		accessCtrl:  accessCtrl,
		authSvc:     authSvc,
		sessionSvc:  sessionSvc,
		consentRepo: consentRepo,
	}
}

func (h *UserInfoHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		mapAuthError(w, err)
		return
	}
	JSON(w, http.StatusOK, userPayload(user))
}

type updateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

func (h *UserInfoHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	var req updateProfileRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		mapAuthError(w, err)
		return
	}
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, userPayload(user))
}

type setAliasRequest struct {
	Alias string `json:"alias"`
}

func (h *UserInfoHandler) SetAlias(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	var req setAliasRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		mapAuthError(w, err)
		return
	}
	if user.Alias != nil && strings.TrimSpace(*user.Alias) != "" {
		Error(w, http.StatusConflict, "alias_already_set", "alias can only be set once")
		return
	}
	if err := h.accessCtrl.ValidateAlias(r.Context(), req.Alias); err != nil {
		Error(w, http.StatusBadRequest, "invalid_alias", err.Error())
		return
	}
	alias := strings.TrimSpace(req.Alias)
	user.Alias = &alias
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, userPayload(user))
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *UserInfoHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	var req changePasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.authSvc.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		mapAuthError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"changed": true})
}

func (h *UserInfoHandler) ListBindings(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	bindings, err := h.socialSvc.ListBindings(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(bindings))
	for _, b := range bindings {
		out = append(out, map[string]any{
			"id":            b.ID,
			"provider":      b.Provider,
			"provider_uid":  b.ProviderUID,
			"provider_name": b.ProviderName,
			"bound_at":      b.BoundAt,
		})
	}
	JSON(w, http.StatusOK, out)
}

func (h *UserInfoHandler) Unbind(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	provider := chi.URLParam(r, "provider")
	if err := h.socialSvc.Unbind(r.Context(), userID, provider); err != nil {
		mapSocialError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"unbound": true})
}

func (h *UserInfoHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	if err := h.authSvc.ResendVerificationEmail(r.Context(), userID); err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (h *UserInfoHandler) SecurityLevel(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	info, err := h.securitySvc.GetLevelInfo(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, info)
}

func (h *UserInfoHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	sessions, err := h.sessionSvc.ListSessions(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, map[string]any{
			"id":         s.ID,
			"ip":         s.IPAddress,
			"user_agent": s.UserAgent,
			"expires_at": s.ExpiresAt,
			"created_at": s.CreatedAt,
		})
	}
	JSON(w, http.StatusOK, out)
}

func (h *UserInfoHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.sessionSvc.RevokeSession(r.Context(), id, userID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		if errors.Is(err, service.ErrPermissionDenied) {
			Error(w, http.StatusForbidden, "forbidden", err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (h *UserInfoHandler) ListAuthorizedApps(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	if h.consentRepo == nil {
		JSON(w, http.StatusOK, []any{})
		return
	}
	apps, err := h.consentRepo.ListAuthorizedApps(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if apps == nil {
		apps = []*domain.UserAuthorization{}
	}
	JSON(w, http.StatusOK, apps)
}

func (h *UserInfoHandler) RevokeAuthorizedApp(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	clientID := chi.URLParam(r, "clientId")
	if clientID == "" {
		Error(w, http.StatusBadRequest, "invalid_request", "client_id is required")
		return
	}
	if h.consentRepo == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "consent repo not available")
		return
	}
	if err := h.consentRepo.DeleteByUserAndClient(r.Context(), userID, clientID); err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func userPayload(u *domain.User) map[string]any {
	return map[string]any{
		"id":             u.ID,
		"uid":            u.UID,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"display_name":   u.DisplayName,
		"alias":          u.Alias,
		"avatar_url":     u.AvatarURL,
		"security_level": u.SecurityLevel,
		"role":           u.Role,
		"status":         u.Status,
		"last_login_at":  u.LastLoginAt,
		"created_at":     u.CreatedAt,
		"updated_at":     u.UpdatedAt,
	}
}
