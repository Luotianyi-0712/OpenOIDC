package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type DeveloperHandler struct {
	clientSvc    *service.ClientService
	riskSvc      *service.RiskService
	settingsRepo port.SettingsRepository
	consentRepo  port.ConsentRepository
}

func NewDeveloperHandler(clientSvc *service.ClientService, riskSvc *service.RiskService, settingsRepo port.SettingsRepository, consentRepo port.ConsentRepository) *DeveloperHandler {
	return &DeveloperHandler{clientSvc: clientSvc, riskSvc: riskSvc, settingsRepo: settingsRepo, consentRepo: consentRepo}
}

// ListApps returns all clients owned by the authenticated user.
func (h *DeveloperHandler) ListApps(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)

	clients, total, err := h.clientSvc.ListClientsByOwner(r.Context(), userID, offset, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		out = append(out, devClientPayload(c))
	}
	PaginatedJSON(w, http.StatusOK, out, total, offset, limit)
}

type devCreateAppRequest struct {
	ClientName       string   `json:"client_name"`
	Description      string   `json:"description"`
	LogoURL          string   `json:"logo_url"`
	RedirectURIs     []string `json:"redirect_uris"`
	Scopes           []string `json:"scopes"`
	GrantTypes       []string `json:"grant_types"`
	MinSecurityLevel *int     `json:"min_security_level"`
}

// CreateApp creates a new OAuth2 client owned by the authenticated user.
func (h *DeveloperHandler) CreateApp(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	var req devCreateAppRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	minLevel := 0
	if req.MinSecurityLevel != nil {
		minLevel = *req.MinSecurityLevel
	}

	input := service.CreateClientInput{
		ClientName:       req.ClientName,
		Description:      req.Description,
		LogoURL:          req.LogoURL,
		OwnerUserID:      &userID,
		RedirectURIs:     req.RedirectURIs,
		GrantTypes:       grantTypes,
		Scopes:           req.Scopes,
		MinSecurityLevel: minLevel,
		ProtocolType:     "oidc",
		IsConfidential:   true,
	}

	client, secret, err := h.clientSvc.CreateClient(r.Context(), input)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	payload := devClientPayload(client)
	payload["client_secret"] = secret
	JSON(w, http.StatusCreated, payload)
}

// GetApp returns a single client owned by the authenticated user, including platform endpoints.
func (h *DeveloperHandler) GetApp(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
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
	if client.OwnerUserID == nil || *client.OwnerUserID != userID {
		Error(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	issuer := h.getIssuer(r.Context())
	payload := devClientPayload(client)
	payload["endpoints"] = map[string]string{
		"authorize_url": issuer + "/oauth2/authorize",
		"token_url":     issuer + "/oauth2/token",
		"userinfo_url":  issuer + "/oauth2/userinfo",
		"jwks_url":      issuer + "/jwks.json",
		"issuer":        issuer,
		"discovery_url": issuer + "/.well-known/openid-configuration",
	}
	// Add user count (number of unique users who have authorized this app).
	if h.consentRepo != nil {
		count, err := h.consentRepo.CountUniqueUsers(r.Context(), client.ClientID)
		if err == nil {
			payload["user_count"] = count
		}
	}
	JSON(w, http.StatusOK, payload)
}

type devUpdateAppRequest struct {
	ClientName       *string  `json:"client_name"`
	Description      *string  `json:"description"`
	LogoURL          *string  `json:"logo_url"`
	RedirectURIs     []string `json:"redirect_uris"`
	Scopes           []string `json:"scopes"`
	GrantTypes       []string `json:"grant_types"`
	MinSecurityLevel *int     `json:"min_security_level"`
}

// UpdateApp updates name, description, logo_url, redirect_uris, and scopes for a caller-owned client.
func (h *DeveloperHandler) UpdateApp(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req devUpdateAppRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	client, err := h.clientSvc.GetClient(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	if client.OwnerUserID == nil || *client.OwnerUserID != userID {
		Error(w, http.StatusNotFound, "not_found", "not found")
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
	if req.Scopes != nil {
		client.Scopes = req.Scopes
	}
	if req.GrantTypes != nil {
		client.GrantTypes = req.GrantTypes
	}
	if req.MinSecurityLevel != nil {
		client.MinSecurityLevel = *req.MinSecurityLevel
	}

	if err := h.clientSvc.UpdateClient(r.Context(), client); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, devClientPayload(client))
}

// DeleteApp deletes a caller-owned client.
func (h *DeveloperHandler) DeleteApp(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
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
	if client.OwnerUserID == nil || *client.OwnerUserID != userID {
		Error(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	if err := h.clientSvc.DeleteClient(r.Context(), id); err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// RotateSecret rotates the client secret for a caller-owned client.
func (h *DeveloperHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	userID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
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
	if client.OwnerUserID == nil || *client.OwnerUserID != userID {
		Error(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	secret, err := h.clientSvc.RotateSecret(r.Context(), id)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{"client_secret": secret})
}

// getIssuer reads the "issuer" setting from the settings repo, with a sensible fallback.
func (h *DeveloperHandler) getIssuer(ctx context.Context) string {
	setting, err := h.settingsRepo.Get(ctx, "issuer")
	if err == nil && setting != nil && strings.TrimSpace(setting.Value) != "" {
		return strings.TrimRight(setting.Value, "/")
	}
	return "http://localhost:8080"
}

type devReportUserRequest struct {
	UserID   string `json:"user_id"`
	Reason   string `json:"reason"`
	Category string `json:"category"`
}

// ReportUser allows a developer to report an abusive user of their app.
func (h *DeveloperHandler) ReportUser(w http.ResponseWriter, r *http.Request) {
	callerID, err := mw.GetUserID(r.Context())
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthenticated", err.Error())
		return
	}
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	client, err := h.clientSvc.GetClient(r.Context(), appID)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	if client.OwnerUserID == nil || *client.OwnerUserID != callerID {
		Error(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	var req devReportUserRequest
	if err := DecodeJSON(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_input", "invalid user_id")
		return
	}

	// Verify the target user has actually authorized this app.
	if h.consentRepo != nil {
		apps, err := h.consentRepo.ListAuthorizedApps(r.Context(), targetID)
		if err == nil {
			found := false
			for _, a := range apps {
				if a.ClientID == client.ClientID {
					found = true
					break
				}
			}
			if !found {
				Error(w, http.StatusForbidden, "not_authorized", "target user has not authorized this app")
				return
			}
		}
	}

	if h.riskSvc == nil {
		Error(w, http.StatusNotImplemented, "not_implemented", "risk service not available")
		return
	}

	report, err := h.riskSvc.ReportUser(r.Context(), appID, callerID, targetID, req.Reason, req.Category)
	if err != nil {
		mapAdminError(w, err)
		return
	}
	JSON(w, http.StatusCreated, report)
}

func devClientPayload(c *domain.OIDCClient) map[string]any {
	return map[string]any{
		"id":                          c.ID,
		"client_id":                   c.ClientID,
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
}
