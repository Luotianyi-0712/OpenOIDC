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
	DisplayName *string            `json:"display_name"`
	Status      *domain.UserStatus `json:"status"`
	Role        *string            `json:"role"`
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

	upd := service.AdminUserUpdate{DisplayName: req.DisplayName, Status: req.Status, Role: req.Role}
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

// ---------------- Clients ----------------

type createClientRequest struct {
	ClientName       string     `json:"client_name"`
	Description      string     `json:"description"`
	LogoURL          string     `json:"logo_url"`
	OwnerUserID      *uuid.UUID `json:"owner_user_id"`
	RedirectURIs     []string   `json:"redirect_uris"`
	GrantTypes       []string   `json:"grant_types"`
	Scopes           []string   `json:"scopes"`
	MinSecurityLevel int        `json:"min_security_level"`
	ProtocolType     string     `json:"protocol_type"`
	IsConfidential   bool       `json:"is_confidential"`
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
		ClientName:       req.ClientName,
		Description:      req.Description,
		LogoURL:          req.LogoURL,
		OwnerUserID:      req.OwnerUserID,
		RedirectURIs:     req.RedirectURIs,
		GrantTypes:       req.GrantTypes,
		Scopes:           req.Scopes,
		MinSecurityLevel: req.MinSecurityLevel,
		ProtocolType:     req.ProtocolType,
		IsConfidential:   req.IsConfidential,
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
	ClientName       *string  `json:"client_name"`
	Description      *string  `json:"description"`
	LogoURL          *string  `json:"logo_url"`
	RedirectURIs     []string `json:"redirect_uris"`
	GrantTypes       []string `json:"grant_types"`
	Scopes           []string `json:"scopes"`
	MinSecurityLevel *int     `json:"min_security_level"`
	IsActive         *bool    `json:"is_active"`
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
	if req.RedirectURIs != nil {
		client.RedirectURIs = req.RedirectURIs
	}
	if req.GrantTypes != nil {
		client.GrantTypes = req.GrantTypes
	}
	if req.Scopes != nil {
		client.Scopes = req.Scopes
	}
	if req.MinSecurityLevel != nil {
		client.MinSecurityLevel = *req.MinSecurityLevel
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

type updateProviderRequest struct {
	Enabled      *bool   `json:"enabled"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	AppID        *string `json:"app_id"`
	AppSecret    *string `json:"app_secret"`
	DisplayName  *string `json:"display_name"`
	TeamID       *string `json:"team_id"`
	KeyID        *string `json:"key_id"`
	PrivateKey   *string `json:"private_key"`
	BaseURL      *string `json:"base_url"`
	TenantID     *string `json:"tenant_id"`
}

func (h *AdminHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	var req updateProviderRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	existing, err := h.adminSvc.GetProvider(r.Context(), providerName)
	if err != nil {
		existing = &domain.ProviderConfig{Provider: providerName}
	}

	if req.Enabled != nil {
		existing.IsEnabled = *req.Enabled
	}
	if req.DisplayName != nil {
		existing.DisplayName = *req.DisplayName
	}
	if req.ClientID != nil {
		existing.ClientID = req.ClientID
	}
	if req.ClientSecret != nil {
		existing.ClientSecret = req.ClientSecret
	}
	if req.AppID != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["app_id"] = *req.AppID
		existing.ClientID = req.AppID
	}
	if req.AppSecret != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["app_secret"] = *req.AppSecret
		existing.ClientSecret = req.AppSecret
	}
	if req.TeamID != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["team_id"] = *req.TeamID
	}
	if req.KeyID != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["key_id"] = *req.KeyID
	}
	if req.PrivateKey != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["private_key"] = *req.PrivateKey
	}
	if req.BaseURL != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["base_url"] = *req.BaseURL
	}
	if req.TenantID != nil {
		if existing.ExtraConfig == nil {
			existing.ExtraConfig = make(map[string]any)
		}
		existing.ExtraConfig["tenant_id"] = *req.TenantID
	}

	if err := h.adminSvc.UpdateProvider(r.Context(), existing); err != nil {
		mapAdminError(w, err)
		return
	}

	if h.socialRegistry != nil {
		if err := h.socialRegistry.Reload(r.Context()); err != nil {
			slog.Warn("reload social registry after provider update", "error", err)
		}
	}

	JSON(w, http.StatusOK, providerPayload(existing))
}

func providerPayload(pc *domain.ProviderConfig) map[string]any {
	m := map[string]any{
		"provider":     pc.Provider,
		"display_name": pc.DisplayName,
		"enabled":      pc.IsEnabled,
		"sort_order":   pc.SortOrder,
		"created_at":   pc.CreatedAt,
		"updated_at":   pc.UpdatedAt,
	}
	if pc.ClientID != nil && *pc.ClientID != "" {
		m["client_id"] = *pc.ClientID
	}
	m["has_secret"] = pc.ClientSecret != nil && *pc.ClientSecret != ""

	if pc.ExtraConfig != nil {
		if appID, ok := pc.ExtraConfig["app_id"].(string); ok && appID != "" {
			m["app_id"] = appID
		}
		if teamID, ok := pc.ExtraConfig["team_id"].(string); ok && teamID != "" {
			m["team_id"] = teamID
		}
		if keyID, ok := pc.ExtraConfig["key_id"].(string); ok && keyID != "" {
			m["key_id"] = keyID
		}
		if baseURL, ok := pc.ExtraConfig["base_url"].(string); ok && baseURL != "" {
			m["base_url"] = baseURL
		}
		if tenantID, ok := pc.ExtraConfig["tenant_id"].(string); ok && tenantID != "" {
			m["tenant_id"] = tenantID
		}
		m["has_app_secret"] = false
		if as, ok := pc.ExtraConfig["app_secret"].(string); ok && as != "" {
			m["has_app_secret"] = true
		}
		m["has_private_key"] = false
		if pk, ok := pc.ExtraConfig["private_key"].(string); ok && pk != "" {
			m["has_private_key"] = true
		}
	}
	return m
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
	// Retrieve password policy settings (with defaults from security config key pattern).
	minLength := 8
	requireUpper := false
	requireLower := false
	requireDigit := false
	requireSymbol := false

	// Try reading from settings repo (dynamically configured).
	if s, err := h.adminSvc.GetSetting(r.Context(), "password_min_length"); err == nil && s.Value != "" {
		if v, e := strconv.Atoi(s.Value); e == nil && v > 0 {
			minLength = v
		}
	}
	if s, err := h.adminSvc.GetSetting(r.Context(), "password_require_upper"); err == nil {
		requireUpper = s.Value == "true"
	}
	if s, err := h.adminSvc.GetSetting(r.Context(), "password_require_lower"); err == nil {
		requireLower = s.Value == "true"
	}
	if s, err := h.adminSvc.GetSetting(r.Context(), "password_require_digit"); err == nil {
		requireDigit = s.Value == "true"
	}
	if s, err := h.adminSvc.GetSetting(r.Context(), "password_require_symbol"); err == nil {
		requireSymbol = s.Value == "true"
	}

	JSON(w, http.StatusOK, map[string]any{
		"min_length":     minLength,
		"require_upper":  requireUpper,
		"require_lower":  requireLower,
		"require_digit":  requireDigit,
		"require_symbol": requireSymbol,
	})
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
					"id":            b.ID,
					"provider":      b.Provider,
					"provider_uid":  b.ProviderUID,
					"provider_name": b.ProviderName,
					"bound_at":      b.BoundAt,
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
		"id":                          c.ID,
		"client_id":                   c.ClientID,
		"client_secret":               c.ClientSecretPlain,
		"client_name":                 c.ClientName,
		"description":                 c.Description,
		"logo_url":                    c.LogoURL,
		"owner_user_id":               c.OwnerUserID,
		"redirect_uris":               c.RedirectURIs,
		"grant_types":                 c.GrantTypes,
		"response_types":              c.ResponseTypes,
		"scopes":                      c.Scopes,
		"token_endpoint_auth_method":  c.TokenEndpointAuthMethod,
		"min_security_level":          c.MinSecurityLevel,
		"require_email_verified":      c.RequireEmailVerified,
		"protocol_type":               c.ProtocolType,
		"is_active":                   c.IsActive,
		"is_confidential":             c.IsConfidential,
		"created_at":                  c.CreatedAt,
		"updated_at":                  c.UpdatedAt,
	}
	if c.OwnerUserID != nil {
		if owner, err := h.userRepo.GetByID(ctx, *c.OwnerUserID); err == nil {
			payload["owner_email"] = owner.Email
			payload["owner_display_name"] = owner.DisplayName
		}
	}
	return payload
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
	case errors.Is(err, service.ErrInvalidAlias):
		Error(w, http.StatusBadRequest, "invalid_alias", err.Error())
	case errors.Is(err, service.ErrPermissionDenied):
		Error(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal", err.Error())
	}
}
