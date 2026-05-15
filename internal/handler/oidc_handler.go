package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/ory/fosite"

	"github.com/anthropic/oidc-platform/internal/config"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/oidcprovider"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type OIDCHandler struct {
	provider   fosite.OAuth2Provider
	userRepo   port.UserRepository
	clientSvc  *service.ClientService
	accessCtrl *service.AccessControlService
	sessionSvc *service.SessionService
	serverCfg  config.ServerConfig
	loginURL   string
}

func NewOIDCHandler(
	provider fosite.OAuth2Provider,
	userRepo port.UserRepository,
	clientSvc *service.ClientService,
	accessCtrl *service.AccessControlService,
	sessionSvc *service.SessionService,
	serverCfg config.ServerConfig,
	loginURL string,
) *OIDCHandler {
	return &OIDCHandler{
		provider:   provider,
		userRepo:   userRepo,
		clientSvc:  clientSvc,
		accessCtrl: accessCtrl,
		sessionSvc: sessionSvc,
		serverCfg:  serverCfg,
		loginURL:   loginURL,
	}
}

// Authorize handles GET /oauth2/authorize.
func (h *OIDCHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ar, err := h.provider.NewAuthorizeRequest(ctx, r)
	if err != nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	userID, err := mw.GetUserID(ctx)
	if err != nil {
		// Not logged in - redirect to login page with return_to.
		redirect := h.loginURL + "?return_to=" + r.URL.RequestURI()
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrServerError.WithWrap(err))
		return
	}

	client, err := h.clientSvc.GetClientByClientID(ctx, ar.GetClient().GetID())
	if err != nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrInvalidClient.WithWrap(err))
		return
	}

	if user.SecurityLevel < client.MinSecurityLevel {
		h.provider.WriteAuthorizeError(ctx, w, ar,
			fosite.ErrAccessDenied.WithHint("security level insufficient for this client"))
		return
	}

	if ok, reason := h.accessCtrl.CheckAccess(ctx, client, user, clientIP(r)); !ok {
		h.provider.WriteAuthorizeError(ctx, w, ar,
			fosite.ErrAccessDenied.WithHint(reason))
		return
	}

	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}
	for _, aud := range ar.GetRequestedAudience() {
		ar.GrantAudience(aud)
	}

	session := oidcprovider.NewSession(user.ID.String(), user.SecurityLevel, h.serverCfg.Issuer, client.ClientID, user.ID.String())
	oidcprovider.AddCustomClaims(session, map[string]any{
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"name":           user.DisplayName,
		"security_level": user.SecurityLevel,
		"alias":          user.Alias,
	})

	response, err := h.provider.NewAuthorizeResponse(ctx, ar, session)
	if err != nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	h.provider.WriteAuthorizeResponse(ctx, w, ar, response)
}

// Token handles POST /oauth2/token.
func (h *OIDCHandler) Token(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := &oidcprovider.Session{}

	accessReq, err := h.provider.NewAccessRequest(ctx, r, session)
	if err != nil {
		h.provider.WriteAccessError(ctx, w, accessReq, err)
		return
	}

	if accessReq.GetGrantTypes().ExactOne("client_credentials") {
		for _, scope := range accessReq.GetRequestedScopes() {
			accessReq.GrantScope(scope)
		}
	}

	response, err := h.provider.NewAccessResponse(ctx, accessReq)
	if err != nil {
		h.provider.WriteAccessError(ctx, w, accessReq, err)
		return
	}

	h.provider.WriteAccessResponse(ctx, w, accessReq, response)
}

// Revoke handles POST /oauth2/revoke.
func (h *OIDCHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.provider.NewRevocationRequest(ctx, r); err != nil {
		h.provider.WriteRevocationResponse(ctx, w, err)
		return
	}
	h.provider.WriteRevocationResponse(ctx, w, nil)
}

// Introspect handles POST /oauth2/introspect.
func (h *OIDCHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := &oidcprovider.Session{}
	resp, err := h.provider.NewIntrospectionRequest(ctx, r, session)
	if err != nil {
		h.provider.WriteIntrospectionError(ctx, w, err)
		return
	}
	h.provider.WriteIntrospectionResponse(ctx, w, resp)
}

// UserInfo handles GET/POST /oauth2/userinfo.
func (h *OIDCHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := &oidcprovider.Session{}
	_, ar, err := h.provider.IntrospectToken(ctx, fosite.AccessTokenFromRequest(r), fosite.AccessToken, session)
	if err != nil {
		Error(w, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}

	userID := session.UserID
	if userID == "" {
		Error(w, http.StatusUnauthorized, "invalid_token", "no user id in token")
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		Error(w, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}

	user, err := h.userRepo.GetByID(ctx, uid)
	if err != nil {
		Error(w, http.StatusNotFound, "user_not_found", err.Error())
		return
	}

	payload := map[string]any{
		"sub":            user.ID.String(),
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"name":           user.DisplayName,
		"avatar_url":     user.AvatarURL,
		"alias":          user.Alias,
		"security_level": user.SecurityLevel,
	}
	_ = ar
	Raw(w, http.StatusOK, payload)
}

// ConsentAccept handles POST /api/v1/consent/accept. For now we use auto-consent in Authorize,
// but this endpoint is provided so a UI flow can call it explicitly.
func (h *OIDCHandler) ConsentAccept(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"accepted": true})
}

// ConsentReject handles POST /api/v1/consent/reject.
func (h *OIDCHandler) ConsentReject(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"rejected": true})
}

