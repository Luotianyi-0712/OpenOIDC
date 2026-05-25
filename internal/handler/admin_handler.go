package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type AdminHandler struct {
	adminSvc       *service.AdminService
	clientSvc      *service.ClientService
	securitySvc    *service.SecurityLevelService
	userRepo       port.UserRepository
	socialRegistry port.SocialProviderRegistry
	riskSvc        *service.RiskService
	sessionRepo    port.SessionRepository
	bindingRepo    port.BindingRepository
	consentRepo    port.ConsentRepository
}

func NewAdminHandler(adminSvc *service.AdminService, clientSvc *service.ClientService, securitySvc *service.SecurityLevelService, userRepo port.UserRepository, socialRegistry port.SocialProviderRegistry, riskSvc *service.RiskService, sessionRepo port.SessionRepository, bindingRepo port.BindingRepository, consentRepo port.ConsentRepository) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, clientSvc: clientSvc, securitySvc: securitySvc, userRepo: userRepo, socialRegistry: socialRegistry, riskSvc: riskSvc, sessionRepo: sessionRepo, bindingRepo: bindingRepo, consentRepo: consentRepo}
}

// ---------------- Users ----------------

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := port.ListUsersOptions{
		Search: q.Get("search"),
		Offset: parseIntDefault(q.Get("offset"), 0),
		Limit:  parseIntDefault(q.Get("limit"), 50),
	}
	if s := q.Get("status"); s != "" {
		st := domain.UserStatus(s)
		opts.Status = &st
	}
	users, total, err := h.adminSvc.ListUsers(r.Context(), opts)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, userPayload(u))
	}
	PaginatedJSON(w, http.StatusOK, out, total, opts.Offset, opts.Limit)
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	user, err := h.adminSvc.GetUser(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, userPayload(user))
}

type adminUserUpdateRequest struct {
	Email         *string            `json:"email"`
	EmailVerified *bool              `json:"email_verified"`
	DisplayName   *string            `json:"display_name"`
	Alias         *string            `json:"alias"`
	AvatarURL     *string            `json:"avatar_url"`
	Status        *domain.UserStatus `json:"status"`
	Role          *string            `json:"role"`
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req adminUserUpdateRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Validate role value if provided.
	if req.Role != nil {
		switch *req.Role {
		case domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleUser:
			// valid
		default:
			Error(w, http.StatusBadRequest, "invalid_input", "role must be one of: user, admin, super_admin")
			return
		}

		// Only super_admin can change roles.
		callerID, err := mw.GetUserID(r.Context())
		if err != nil {
			Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
			return
		}
		caller, err := h.userRepo.GetByID(r.Context(), callerID)
		if err != nil {
			Error(w, http.StatusInternalServerError, "internal", "failed to look up requesting user")
			return
		}
		if !caller.IsSuperAdmin() {
			Error(w, http.StatusForbidden, "forbidden", "only super_admin can change user roles")
			return
		}
	}

	upd := service.AdminUserUpdate{
		Email:         req.Email,
		EmailVerified: req.EmailVerified,
		DisplayName:   req.DisplayName,
		Alias:         req.Alias,
		AvatarURL:     req.AvatarURL,
		Status:        req.Status,
		Role:          req.Role,
	}
	if err := h.adminSvc.UpdateUser(r.Context(), id, upd); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.adminSvc.DeleteUser(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

type adminCreateUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req adminCreateUserRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.Email == "" || req.Password == "" {
		Error(w, http.StatusBadRequest, "invalid_input", "email and password are required")
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Role == domain.RoleAdmin || req.Role == domain.RoleSuperAdmin {
		callerID, err := mw.GetUserID(r.Context())
		if err != nil {
			Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
			return
		}
		caller, err := h.userRepo.GetByID(r.Context(), callerID)
		if err != nil {
			Error(w, http.StatusInternalServerError, "internal", "failed to look up requesting user")
			return
		}
		if !caller.IsSuperAdmin() {
			Error(w, http.StatusForbidden, "forbidden", "only super_admin can create admin users")
			return
		}
	}

	user, err := h.adminSvc.CreateUser(r.Context(), req.Email, req.Password, req.DisplayName, req.Role)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusCreated, userPayload(user))
}

type overrideLevelRequest struct {
	Level int `json:"level"`
}

func (h *AdminHandler) OverrideSecurityLevel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req overrideLevelRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.adminSvc.OverrideSecurityLevel(r.Context(), id, req.Level); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"updated": true})
}

type resetUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func (h *AdminHandler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req resetUserPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.NewPassword == "" {
		Error(w, http.StatusBadRequest, "invalid_input", "new_password is required")
		return
	}
	if err := h.adminSvc.ResetUserPassword(r.Context(), id, req.NewPassword); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"reset": true})
}

func (h *AdminHandler) RevokeUserSession(w http.ResponseWriter, r *http.Request) {
	if h.sessionRepo == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "session repository not available")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_session_id", err.Error())
		return
	}
	sessions, err := h.sessionRepo.ListByUser(r.Context(), id)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	found := false
	for _, session := range sessions {
		if session.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		Error(w, http.StatusNotFound, "not_found", "session not found for user")
		return
	}
	if err := h.sessionRepo.Delete(r.Context(), sessionID); err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (h *AdminHandler) UnbindUserSocial(w http.ResponseWriter, r *http.Request) {
	if h.bindingRepo == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "binding repository not available")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	if provider == "" {
		Error(w, http.StatusBadRequest, "invalid_provider", "provider is required")
		return
	}
	if err := h.bindingRepo.SoftUnbind(r.Context(), id, provider, "admin_unbound"); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"unbound": true})
}

func (h *AdminHandler) ListUserPasskeys(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	passkeys, err := h.adminSvc.ListUserPasskeys(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(passkeys))
	for _, passkey := range passkeys {
		out = append(out, adminPasskeyPayload(passkey))
	}
	JSON(w, http.StatusOK, out)
}

func (h *AdminHandler) DeleteUserPasskey(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	passkeyID, err := uuid.Parse(chi.URLParam(r, "passkey_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_passkey_id", err.Error())
		return
	}
	if err := h.adminSvc.DeleteUserPasskey(r.Context(), id, passkeyID); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------------- Clients ----------------

type createClientRequest struct {
	ClientName              string     `json:"client_name"`
	Description             string     `json:"description"`
	LogoURL                 string     `json:"logo_url"`
	HomepageURL             string     `json:"homepage_url"`
	OwnerUserID             *uuid.UUID `json:"owner_user_id"`
	RedirectURIs            []string   `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string   `json:"post_logout_redirect_uris"`
	GrantTypes              []string   `json:"grant_types"`
	ResponseTypes           []string   `json:"response_types"`
	Scopes                  []string   `json:"scopes"`
	TokenEndpointAuthMethod string     `json:"token_endpoint_auth_method"`
	MinSecurityLevel        int        `json:"min_security_level"`
	RequireEmailVerified    *bool      `json:"require_email_verified"`
	ProtocolType            string     `json:"protocol_type"`
	IsConfidential          bool       `json:"is_confidential"`
}

func (h *AdminHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	clients, total, err := h.clientSvc.ListClients(r.Context(), offset, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		out = append(out, h.clientPayload(r.Context(), c))
	}
	PaginatedJSON(w, http.StatusOK, out, total, offset, limit)
}

func (h *AdminHandler) ListUserClients(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if _, err := h.adminSvc.GetUser(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}

	clients, total, err := h.clientSvc.ListClientsByOwner(r.Context(), id, 0, 100)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		out = append(out, h.clientPayload(r.Context(), c))
	}
	PaginatedJSON(w, http.StatusOK, out, total, 0, 100)
}

func (h *AdminHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req createClientRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input := service.CreateClientInput{
		ClientName:              req.ClientName,
		Description:             req.Description,
		LogoURL:                 req.LogoURL,
		HomepageURL:             req.HomepageURL,
		OwnerUserID:             req.OwnerUserID,
		RedirectURIs:            req.RedirectURIs,
		PostLogoutRedirectURIs:  req.PostLogoutRedirectURIs,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		Scopes:                  req.Scopes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		MinSecurityLevel:        req.MinSecurityLevel,
		RequireEmailVerified:    req.RequireEmailVerified,
		ProtocolType:            req.ProtocolType,
		IsConfidential:          req.IsConfidential,
	}
	client, secret, err := h.clientSvc.CreateClient(r.Context(), input)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	payload := h.clientPayload(r.Context(), client)
	payload["client_secret"] = secret
	JSON(w, http.StatusCreated, payload)
}

func (h *AdminHandler) GetClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	client, err := h.clientSvc.GetClient(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, h.clientPayload(r.Context(), client))
}

type updateClientRequest struct {
	ClientName              *string  `json:"client_name"`
	Description             *string  `json:"description"`
	LogoURL                 *string  `json:"logo_url"`
	HomepageURL             *string  `json:"homepage_url"`
	RedirectURIs            []string `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scopes                  []string `json:"scopes"`
	TokenEndpointAuthMethod *string  `json:"token_endpoint_auth_method"`
	MinSecurityLevel        *int     `json:"min_security_level"`
	RequireEmailVerified    *bool    `json:"require_email_verified"`
	ProtocolType            *string  `json:"protocol_type"`
	IsConfidential          *bool    `json:"is_confidential"`
	IsActive                *bool    `json:"is_active"`
}

func (h *AdminHandler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req updateClientRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	client, err := h.clientSvc.GetClient(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	if req.ClientName != nil {
		client.ClientName = *req.ClientName
	}
	if req.Description != nil {
		client.Description = *req.Description
	}
	if req.LogoURL != nil {
		client.LogoURL = *req.LogoURL
	}
	if req.HomepageURL != nil {
		client.HomepageURL = *req.HomepageURL
	}
	if req.RedirectURIs != nil {
		client.RedirectURIs = req.RedirectURIs
	}
	if req.PostLogoutRedirectURIs != nil {
		client.PostLogoutRedirectURIs = req.PostLogoutRedirectURIs
	}
	if req.GrantTypes != nil {
		client.GrantTypes = req.GrantTypes
	}
	if req.ResponseTypes != nil {
		client.ResponseTypes = req.ResponseTypes
	}
	if req.Scopes != nil {
		client.Scopes = req.Scopes
	}
	if req.TokenEndpointAuthMethod != nil {
		client.TokenEndpointAuthMethod = *req.TokenEndpointAuthMethod
	}
	if req.MinSecurityLevel != nil {
		client.MinSecurityLevel = *req.MinSecurityLevel
	}
	if req.RequireEmailVerified != nil {
		client.RequireEmailVerified = *req.RequireEmailVerified
	}
	if req.ProtocolType != nil {
		client.ProtocolType = *req.ProtocolType
	}
	if req.IsConfidential != nil {
		client.IsConfidential = *req.IsConfidential
		if !client.IsConfidential {
			client.TokenEndpointAuthMethod = "none"
		} else if client.TokenEndpointAuthMethod == "" || client.TokenEndpointAuthMethod == "none" {
			client.TokenEndpointAuthMethod = "client_secret_basic"
		}
	}
	if req.IsActive != nil {
		client.IsActive = *req.IsActive
	}
	if err := h.clientSvc.UpdateClient(r.Context(), client); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, h.clientPayload(r.Context(), client))
}

func (h *AdminHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.clientSvc.DeleteClient(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminHandler) RotateClientSecret(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	secret, err := h.clientSvc.RotateSecret(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"client_secret": secret})
}

type accessRuleRequest struct {
	RuleType string `json:"rule_type"`
	Value    string `json:"value"`
}

func (h *AdminHandler) AddClientAccessRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req accessRuleRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	rule, err := h.clientSvc.AddAccessRule(r.Context(), id, req.RuleType, req.Value)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusCreated, rule)
}

func (h *AdminHandler) ListClientAccessRules(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	rules, err := h.clientSvc.ListAccessRules(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, rules)
}

func (h *AdminHandler) RemoveClientAccessRule(w http.ResponseWriter, r *http.Request) {
	rid, err := uuid.Parse(chi.URLParam(r, "rid"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.clientSvc.RemoveAccessRule(r.Context(), rid); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------------- Security Rules ----------------

func (h *AdminHandler) ListSecurityRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.securitySvc.ListRules(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, rules)
}

func (h *AdminHandler) CreateSecurityRule(w http.ResponseWriter, r *http.Request) {
	var rule domain.SecurityLevelRule
	if err := DecodeJSON(r, &rule); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.securitySvc.CreateRule(r.Context(), &rule); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusCreated, rule)
}

func (h *AdminHandler) GetSecurityRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	rule, err := h.securitySvc.GetRule(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, rule)
}

func (h *AdminHandler) UpdateSecurityRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var rule domain.SecurityLevelRule
	if err := DecodeJSON(r, &rule); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	rule.ID = id
	if err := h.securitySvc.UpdateRule(r.Context(), &rule); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, rule)
}

func (h *AdminHandler) DeleteSecurityRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.securitySvc.DeleteRule(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminHandler) RecomputeSecurityLevels(w http.ResponseWriter, r *http.Request) {
	if err := h.securitySvc.RecomputeAll(r.Context()); err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"recomputed": true})
}

// ---------------- Providers ----------------

func (h *AdminHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.adminSvc.ListProviders(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(providers))
	for _, pc := range providers {
		out = append(out, providerPayload(pc))
	}
	JSON(w, http.StatusOK, out)
}

func (h *AdminHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	providerName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	pc, err := h.adminSvc.GetProvider(r.Context(), providerName)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, providerPayload(pc))
}

type updateProviderRequest struct {
	Enabled               *bool    `json:"enabled"`
	ClientID              *string  `json:"client_id"`
	ClientSecret          *string  `json:"client_secret"`
	AppID                 *string  `json:"app_id"`
	AppSecret             *string  `json:"app_secret"`
	DisplayName           *string  `json:"display_name"`
	Type                  *string  `json:"type"`
	TeamID                *string  `json:"team_id"`
	KeyID                 *string  `json:"key_id"`
	PrivateKey            *string  `json:"private_key"`
	BaseURL               *string  `json:"base_url"`
	TenantID              *string  `json:"tenant_id"`
	AuthURL               *string  `json:"auth_url"`
	AuthorizationEndpoint *string  `json:"authorization_endpoint"`
	TokenURL              *string  `json:"token_url"`
	TokenEndpoint         *string  `json:"token_endpoint"`
	UserInfoURL           *string  `json:"userinfo_url"`
	UserInfoEndpoint      *string  `json:"userinfo_endpoint"`
	Scopes                []string `json:"scopes"`
	UserIDPath            *string  `json:"user_id_path"`
	UserIDField           *string  `json:"user_id_field"`
	EmailPath             *string  `json:"email_path"`
	EmailField            *string  `json:"email_field"`
	NamePath              *string  `json:"name_path"`
	NameField             *string  `json:"name_field"`
	AvatarPath            *string  `json:"avatar_path"`
	AvatarField           *string  `json:"avatar_field"`
	IconURL               *string  `json:"icon_url"`
	SortOrder             *int     `json:"sort_order"`
}

type createProviderRequest struct {
	Provider string `json:"provider"`
	updateProviderRequest
}

func (h *AdminHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req createProviderRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	pc := &domain.ProviderConfig{
		Provider:    strings.ToLower(strings.TrimSpace(req.Provider)),
		IsEnabled:   false,
		ExtraConfig: map[string]any{"type": domain.ProviderTypeCustomOAuth2},
	}
	applyProviderRequest(pc, req.updateProviderRequest)

	if err := h.adminSvc.CreateProvider(r.Context(), pc); err != nil {
		mapAdminError(w, err)
		return
	}

	h.reloadSocialRegistry(r.Context())
	JSON(w, http.StatusCreated, providerPayload(pc))
}

func (h *AdminHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	providerName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	var req updateProviderRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	existing, err := h.adminSvc.GetProvider(r.Context(), providerName)
	if err != nil {
		if domain.IsValidCustomProviderKey(providerName) {
			mapAdminError(w, err)
			return
		}
		existing = &domain.ProviderConfig{Provider: providerName}
	}
	applyProviderRequest(existing, req)

	if err := h.adminSvc.UpdateProvider(r.Context(), existing); err != nil {
		mapAdminError(w, err)
		return
	}

	h.reloadSocialRegistry(r.Context())
	JSON(w, http.StatusOK, providerPayload(existing))
}

func (h *AdminHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	providerName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	if err := h.adminSvc.DeleteProvider(r.Context(), providerName); err != nil {
		mapAdminError(w, err)
		return
	}
	h.reloadSocialRegistry(r.Context())
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *AdminHandler) reloadSocialRegistry(ctx context.Context) {
	if h.socialRegistry == nil {
		return
	}
	if err := h.socialRegistry.Reload(ctx); err != nil {
		slog.Warn("reload social registry after provider change", "error", err)
	}
}

func applyProviderRequest(pc *domain.ProviderConfig, req updateProviderRequest) {
	if req.Enabled != nil {
		pc.IsEnabled = *req.Enabled
	}
	if req.DisplayName != nil {
		pc.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.ClientID != nil {
		v := strings.TrimSpace(*req.ClientID)
		pc.ClientID = &v
	}
	if req.ClientSecret != nil {
		v := strings.TrimSpace(*req.ClientSecret)
		pc.ClientSecret = &v
	}
	if req.SortOrder != nil {
		pc.SortOrder = *req.SortOrder
	}
	if req.AppID != nil {
		setProviderExtra(pc, "app_id", *req.AppID)
		v := strings.TrimSpace(*req.AppID)
		pc.ClientID = &v
	}
	if req.AppSecret != nil {
		setProviderExtra(pc, "app_secret", *req.AppSecret)
		v := strings.TrimSpace(*req.AppSecret)
		pc.ClientSecret = &v
	}
	if req.TeamID != nil {
		setProviderExtra(pc, "team_id", *req.TeamID)
	}
	if req.KeyID != nil {
		setProviderExtra(pc, "key_id", *req.KeyID)
	}
	if req.PrivateKey != nil {
		setProviderExtra(pc, "private_key", *req.PrivateKey)
	}
	if req.BaseURL != nil {
		setProviderExtra(pc, "base_url", *req.BaseURL)
	}
	if req.TenantID != nil {
		setProviderExtra(pc, "tenant_id", *req.TenantID)
	}
	if req.AuthURL != nil {
		setProviderExtra(pc, "auth_url", *req.AuthURL)
		setProviderExtra(pc, "authorization_endpoint", *req.AuthURL)
	}
	if req.AuthorizationEndpoint != nil {
		setProviderExtra(pc, "authorization_endpoint", *req.AuthorizationEndpoint)
		setProviderExtra(pc, "auth_url", *req.AuthorizationEndpoint)
	}
	if req.TokenURL != nil {
		setProviderExtra(pc, "token_url", *req.TokenURL)
		setProviderExtra(pc, "token_endpoint", *req.TokenURL)
	}
	if req.TokenEndpoint != nil {
		setProviderExtra(pc, "token_endpoint", *req.TokenEndpoint)
		setProviderExtra(pc, "token_url", *req.TokenEndpoint)
	}
	if req.UserInfoURL != nil {
		setProviderExtra(pc, "userinfo_url", *req.UserInfoURL)
		setProviderExtra(pc, "userinfo_endpoint", *req.UserInfoURL)
	}
	if req.UserInfoEndpoint != nil {
		setProviderExtra(pc, "userinfo_endpoint", *req.UserInfoEndpoint)
		setProviderExtra(pc, "userinfo_url", *req.UserInfoEndpoint)
	}
	if req.UserIDPath != nil {
		setProviderExtra(pc, "user_id_path", *req.UserIDPath)
		setProviderExtra(pc, "user_id_field", *req.UserIDPath)
	}
	if req.UserIDField != nil {
		setProviderExtra(pc, "user_id_field", *req.UserIDField)
		setProviderExtra(pc, "user_id_path", *req.UserIDField)
	}
	if req.EmailPath != nil {
		setProviderExtra(pc, "email_path", *req.EmailPath)
		setProviderExtra(pc, "email_field", *req.EmailPath)
	}
	if req.EmailField != nil {
		setProviderExtra(pc, "email_field", *req.EmailField)
		setProviderExtra(pc, "email_path", *req.EmailField)
	}
	if req.NamePath != nil {
		setProviderExtra(pc, "name_path", *req.NamePath)
		setProviderExtra(pc, "name_field", *req.NamePath)
	}
	if req.NameField != nil {
		setProviderExtra(pc, "name_field", *req.NameField)
		setProviderExtra(pc, "name_path", *req.NameField)
	}
	if req.AvatarPath != nil {
		setProviderExtra(pc, "avatar_path", *req.AvatarPath)
		setProviderExtra(pc, "avatar_field", *req.AvatarPath)
	}
	if req.AvatarField != nil {
		setProviderExtra(pc, "avatar_field", *req.AvatarField)
		setProviderExtra(pc, "avatar_path", *req.AvatarField)
	}
	if req.IconURL != nil {
		setProviderExtra(pc, "icon_url", *req.IconURL)
	}
	if req.Scopes != nil {
		setProviderExtra(pc, "scopes", cleanProviderStringSlice(req.Scopes))
	}
}

func setProviderExtra(pc *domain.ProviderConfig, key string, value any) {
	if pc.ExtraConfig == nil {
		pc.ExtraConfig = make(map[string]any)
	}
	if s, ok := value.(string); ok {
		pc.ExtraConfig[key] = strings.TrimSpace(s)
		return
	}
	pc.ExtraConfig[key] = value
}

func cleanProviderStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func providerPayload(pc *domain.ProviderConfig) map[string]any {
	m := map[string]any{
		"provider":     pc.Provider,
		"display_name": pc.DisplayName,
		"enabled":      pc.IsEnabled,
		"type":         domain.ProviderType(pc),
		"sort_order":   pc.SortOrder,
		"created_at":   pc.CreatedAt,
		"updated_at":   pc.UpdatedAt,
	}
	if pc.ClientID != nil && *pc.ClientID != "" {
		m["client_id"] = *pc.ClientID
	}
	m["has_secret"] = pc.ClientSecret != nil && *pc.ClientSecret != ""

	if pc.ExtraConfig != nil {
		if appID := providerExtraString(pc.ExtraConfig, "app_id"); appID != "" {
			m["app_id"] = appID
		}
		if teamID := providerExtraString(pc.ExtraConfig, "team_id"); teamID != "" {
			m["team_id"] = teamID
		}
		if keyID := providerExtraString(pc.ExtraConfig, "key_id"); keyID != "" {
			m["key_id"] = keyID
		}
		if baseURL := providerExtraString(pc.ExtraConfig, "base_url"); baseURL != "" {
			m["base_url"] = baseURL
		}
		if tenantID := providerExtraString(pc.ExtraConfig, "tenant_id"); tenantID != "" {
			m["tenant_id"] = tenantID
		}
		if iconURL := providerExtraString(pc.ExtraConfig, "icon_url"); iconURL != "" {
			m["icon_url"] = iconURL
		}
		if authEndpoint := firstProviderExtraString(pc.ExtraConfig, "authorization_endpoint", "auth_url"); authEndpoint != "" {
			m["authorization_endpoint"] = authEndpoint
			m["auth_url"] = authEndpoint
		}
		if tokenEndpoint := firstProviderExtraString(pc.ExtraConfig, "token_endpoint", "token_url"); tokenEndpoint != "" {
			m["token_endpoint"] = tokenEndpoint
			m["token_url"] = tokenEndpoint
		}
		if userinfoEndpoint := firstProviderExtraString(pc.ExtraConfig, "userinfo_endpoint", "userinfo_url", "user_url"); userinfoEndpoint != "" {
			m["userinfo_endpoint"] = userinfoEndpoint
			m["userinfo_url"] = userinfoEndpoint
		}
		if userIDField := firstProviderExtraString(pc.ExtraConfig, "user_id_field", "user_id_path"); userIDField != "" {
			m["user_id_field"] = userIDField
			m["user_id_path"] = userIDField
		}
		if emailField := firstProviderExtraString(pc.ExtraConfig, "email_field", "email_path"); emailField != "" {
			m["email_field"] = emailField
			m["email_path"] = emailField
		}
		if nameField := firstProviderExtraString(pc.ExtraConfig, "name_field", "name_path"); nameField != "" {
			m["name_field"] = nameField
			m["name_path"] = nameField
		}
		if avatarField := firstProviderExtraString(pc.ExtraConfig, "avatar_field", "avatar_path"); avatarField != "" {
			m["avatar_field"] = avatarField
			m["avatar_path"] = avatarField
		}
		if scopes := providerExtraStringSlice(pc.ExtraConfig, "scopes"); len(scopes) > 0 {
			m["scopes"] = scopes
		}
		m["has_app_secret"] = providerExtraString(pc.ExtraConfig, "app_secret") != ""
		m["has_private_key"] = providerExtraString(pc.ExtraConfig, "private_key") != ""
	}
	return m
}

func providerExtraString(extra map[string]any, key string) string {
	v, _ := extra[key].(string)
	return strings.TrimSpace(v)
}

func firstProviderExtraString(extra map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := providerExtraString(extra, key); value != "" {
			return value
		}
	}
	return ""
}

func providerExtraStringSlice(extra map[string]any, key string) []string {
	v, ok := extra[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return cleanProviderStringSlice(t)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return cleanProviderStringSlice(out)
	case string:
		parts := strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
		return cleanProviderStringSlice(parts)
	default:
		return nil
	}
}

func maskSecret(s string, showLast int) string {
	if len(s) <= showLast {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-showLast) + s[len(s)-showLast:]
}

// ---------------- Alias restrictions ----------------

type aliasRestrictionRequest struct {
	Pattern         string `json:"pattern"`
	RestrictionType string `json:"restriction_type"`
	Reason          string `json:"reason"`
}

func (h *AdminHandler) ListAliasRestrictions(w http.ResponseWriter, r *http.Request) {
	rs, err := h.adminSvc.ListAliasRestrictions(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, rs)
}

func (h *AdminHandler) CreateAliasRestriction(w http.ResponseWriter, r *http.Request) {
	var req aliasRestrictionRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	res, err := h.adminSvc.CreateAliasRestriction(r.Context(), req.Pattern, req.RestrictionType, req.Reason)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusCreated, res)
}

func (h *AdminHandler) DeleteAliasRestriction(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.adminSvc.DeleteAliasRestriction(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------------- Settings ----------------

// PublicSettings returns login/registration settings without requiring auth.
func (h *AdminHandler) PublicSettings(w http.ResponseWriter, r *http.Request) {
	keys := []string{
		"registration_enabled",
		"registration_email_verification_required",
		"password_login_enabled",
		"social_login_enabled",
		"social_register_enabled",
		"turnstile_site_key",
		"developer_min_trust_level",
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		setting, err := h.adminSvc.GetSetting(r.Context(), key)
		if err != nil {
			if key == "turnstile_site_key" || key == "developer_min_trust_level" {
				result[key] = ""
			} else {
				result[key] = "true"
			}
			continue
		}
		result[key] = setting.Value
	}
	JSON(w, http.StatusOK, result)
}

// PasswordPolicy returns the password requirements for display on registration/change forms.
func (h *AdminHandler) PasswordPolicy(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, h.adminSvc.PasswordPolicy(r.Context()))
}

type updateSettingRequest struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

func (h *AdminHandler) ListSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.adminSvc.ListSettings(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req updateSettingRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.adminSvc.UpdateSetting(r.Context(), key, req.Value, req.Description); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// ---------------- Audit logs ----------------

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := port.ListAuditOptions{
		Action: q.Get("action"),
		Offset: parseIntDefault(q.Get("offset"), 0),
		Limit:  parseIntDefault(q.Get("limit"), 50),
	}
	if uid := q.Get("user_id"); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			opts.UserID = &id
		}
	}
	logs, total, err := h.adminSvc.ListAuditLogs(r.Context(), opts)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		out = append(out, h.auditPayload(r.Context(), log))
	}
	PaginatedJSON(w, http.StatusOK, out, total, opts.Offset, opts.Limit)
}

// ---------------- Signing keys ----------------

func (h *AdminHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.adminSvc.ListSigningKeys(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{
			"id":         k.ID,
			"key_id":     k.KeyID,
			"algorithm":  k.Algorithm,
			"is_current": k.IsCurrent,
			"created_at": k.CreatedAt,
			"rotated_at": k.RotatedAt,
		})
	}
	JSON(w, http.StatusOK, out)
}

func (h *AdminHandler) RotateKey(w http.ResponseWriter, r *http.Request) {
	k, err := h.adminSvc.RotateSigningKey(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"id":     k.ID,
		"key_id": k.KeyID,
	})
}

// ---------------- Risk ----------------

func (h *AdminHandler) ListRiskReports(w http.ResponseWriter, r *http.Request) {
	if h.riskSvc == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "risk service not available")
		return
	}
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	reports, total, err := h.riskSvc.ListPendingReports(r.Context(), offset, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	PaginatedJSON(w, http.StatusOK, reports, total, offset, limit)
}

type confirmReportRequest struct {
	Note string `json:"note"`
}

func (h *AdminHandler) ConfirmRiskReport(w http.ResponseWriter, r *http.Request) {
	if h.riskSvc == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "risk service not available")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	adminID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	var req confirmReportRequest
	_ = DecodeJSON(r, &req)
	if err := h.riskSvc.ConfirmReport(r.Context(), id, adminID, req.Note); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"confirmed": true})
}

func (h *AdminHandler) DismissRiskReport(w http.ResponseWriter, r *http.Request) {
	if h.riskSvc == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "risk service not available")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	adminID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	var req confirmReportRequest
	_ = DecodeJSON(r, &req)
	if err := h.riskSvc.DismissReport(r.Context(), id, adminID, req.Note); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"dismissed": true})
}

func (h *AdminHandler) ListRiskList(w http.ResponseWriter, r *http.Request) {
	if h.riskSvc == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "risk service not available")
		return
	}
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	entries, total, err := h.riskSvc.ListRiskEntries(r.Context(), offset, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	PaginatedJSON(w, http.StatusOK, entries, total, offset, limit)
}

func (h *AdminHandler) RemoveRiskEntry(w http.ResponseWriter, r *http.Request) {
	if h.riskSvc == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "risk service not available")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := h.riskSvc.RemoveFromRiskList(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------------- User Detail ----------------

func (h *AdminHandler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	user, err := h.adminSvc.GetUser(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}

	payload := userPayload(user)

	// Sessions
	if h.sessionRepo != nil {
		sessions, err := h.sessionRepo.ListByUser(r.Context(), id)
		if err == nil {
			sessOut := make([]map[string]any, 0, len(sessions))
			for _, s := range sessions {
				sessOut = append(sessOut, map[string]any{
					"id":         s.ID,
					"ip":         s.IPAddress,
					"user_agent": s.UserAgent,
					"created_at": s.CreatedAt,
					"expires_at": s.ExpiresAt,
				})
			}
			payload["sessions"] = sessOut
		}
	}

	// Bindings
	if h.bindingRepo != nil {
		bindings, err := h.bindingRepo.ListByUser(r.Context(), id)
		if err == nil {
			bindOut := make([]map[string]any, 0, len(bindings))
			for _, b := range bindings {
				bindOut = append(bindOut, map[string]any{
					"id":                 b.ID,
					"provider":           b.Provider,
					"provider_uid":       b.ProviderUID,
					"provider_email":     b.ProviderEmail,
					"provider_name":      b.ProviderName,
					"provider_avatar":    b.ProviderAvatar,
					"status":             b.Status,
					"bound_at":           b.BoundAt,
					"unbound_at":         b.UnboundAt,
					"last_auth_status":   b.LastAuthStatus,
					"last_auth_check_at": b.LastAuthCheckAt,
					"last_auth_error":    b.LastAuthError,
				})
			}
			payload["bindings"] = bindOut
		}
	}

	// Risk reports
	if h.riskSvc != nil {
		reports, err := h.riskSvc.ListReportsByTarget(r.Context(), id)
		if err == nil {
			payload["risk_reports"] = reports
		}
	}

	logs, _, err := h.adminSvc.ListAuditLogs(r.Context(), port.ListAuditOptions{UserID: &id, Limit: 50})
	if err == nil {
		out := make([]map[string]any, 0, len(logs))
		for _, log := range logs {
			out = append(out, h.auditPayload(r.Context(), log))
		}
		payload["audit_logs"] = out
	}

	JSON(w, http.StatusOK, payload)
}

// ---------------- Stats ----------------

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	// Get total users count using limit=1 pagination.
	_, totalUsers, err := h.adminSvc.ListUsers(r.Context(), port.ListUsersOptions{Offset: 0, Limit: 1})
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	// Get total clients count using limit=1 pagination.
	_, totalClients, err := h.clientSvc.ListClients(r.Context(), 0, 1)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	// Get active sessions count.
	var totalSessions int64
	if h.sessionRepo != nil {
		totalSessions, _ = h.sessionRepo.CountActive(r.Context())
	}

	// Get latest 5 audit log entries.
	recentEvents, _, err := h.adminSvc.ListAuditLogs(r.Context(), port.ListAuditOptions{Offset: 0, Limit: 5})
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	recent := make([]map[string]any, 0, len(recentEvents))
	for _, event := range recentEvents {
		recent = append(recent, h.auditPayload(r.Context(), event))
	}

	JSON(w, http.StatusOK, map[string]any{
		"total_users":    totalUsers,
		"total_clients":  totalClients,
		"total_sessions": totalSessions,
		"recent_events":  recent,
	})
}

// ---------------- Helpers ----------------

func (h *AdminHandler) auditPayload(ctx context.Context, log *domain.AuditLog) map[string]any {
	payload := map[string]any{
		"id":            log.ID,
		"user_id":       log.UserID,
		"action":        log.Action,
		"resource_type": log.ResourceType,
		"resource_id":   log.ResourceID,
		"ip_address":    log.IPAddress,
		"user_agent":    log.UserAgent,
		"details":       log.Details,
		"details_text":  formatAuditDetails(log.Details),
		"created_at":    log.CreatedAt,
	}
	if log.UserID != nil {
		if user, err := h.userRepo.GetByID(ctx, *log.UserID); err == nil {
			payload["user_email"] = user.Email
			payload["user_display_name"] = user.DisplayName
		}
	}
	return payload
}

func (h *AdminHandler) clientPayload(ctx context.Context, c *domain.OIDCClient) map[string]any {
	payload := map[string]any{
		"id":                         c.ID,
		"client_id":                  c.ClientID,
		"client_secret":              c.ClientSecretPlain,
		"client_name":                c.ClientName,
		"description":                c.Description,
		"logo_url":                   c.LogoURL,
		"homepage_url":               c.HomepageURL,
		"owner_user_id":              c.OwnerUserID,
		"redirect_uris":              c.RedirectURIs,
		"post_logout_redirect_uris":  c.PostLogoutRedirectURIs,
		"grant_types":                c.GrantTypes,
		"response_types":             c.ResponseTypes,
		"scopes":                     c.Scopes,
		"token_endpoint_auth_method": c.TokenEndpointAuthMethod,
		"min_security_level":         c.MinSecurityLevel,
		"require_email_verified":     c.RequireEmailVerified,
		"protocol_type":              c.ProtocolType,
		"is_active":                  c.IsActive,
		"is_confidential":            c.IsConfidential,
		"created_at":                 c.CreatedAt,
		"updated_at":                 c.UpdatedAt,
	}
	if c.OwnerUserID != nil {
		if owner, err := h.userRepo.GetByID(ctx, *c.OwnerUserID); err == nil {
			payload["owner_email"] = owner.Email
			payload["owner_display_name"] = owner.DisplayName
		}
	}
	return payload
}

func adminPasskeyPayload(c *domain.PasskeyCredential) map[string]any {
	return map[string]any{
		"id":           c.ID,
		"name":         c.Name,
		"created_at":   c.CreatedAt,
		"last_used_at": c.LastUsedAt,
		"transports":   c.Transport,
	}
}

func formatAuditDetails(details map[string]any) string {
	if len(details) == 0 {
		return ""
	}
	parts := make([]string, 0, len(details))
	for key, value := range details {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	return strings.Join(parts, ", ")
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func mapAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound), errors.Is(err, port.ErrNotFound):
		Error(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, service.ErrAlreadyExists), errors.Is(err, port.ErrAlreadyExists):
		Error(w, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		Error(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, service.ErrPasswordTooWeak):
		Error(w, http.StatusBadRequest, "password_too_weak", err.Error())
	case errors.Is(err, service.ErrInvalidAlias):
		Error(w, http.StatusBadRequest, "invalid_alias", err.Error())
	case errors.Is(err, service.ErrPermissionDenied):
		Error(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal", err.Error())
	}
}
