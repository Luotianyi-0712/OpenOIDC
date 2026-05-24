package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

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
	cache      port.Cache
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

// SetCache sets the cache dependency for storing consent challenges.
func (h *OIDCHandler) SetCache(cache port.Cache) {
	h.cache = cache
}

// consentChallenge stores the OIDC authorize request info so we can resume after user consent.
type consentChallenge struct {
	UserID      string   `json:"user_id"`
	ClientID    string   `json:"client_id"`
	ClientName  string   `json:"client_name"`
	Scopes      []string `json:"scopes"`
	RedirectURI string   `json:"redirect_uri"`
	RequestURL  string   `json:"request_url"`
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
		redirect := h.loginURL + "?return_to=" + url.QueryEscape(r.URL.RequestURI())
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

	// If a consent UI is needed, redirect to it with a challenge token.
	// For now, we use auto-consent (skip the consent screen) since the ConsentAccept
	// endpoint can be called by the frontend if needed.
	// Auto-consent: grant all requested scopes and issue response.
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

// ConsentAccept handles POST /api/v1/consent/accept.
// Stores consent and returns a redirect URI if a challenge was provided.
func (h *OIDCHandler) ConsentAccept(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConsentChallenge string `json:"consent_challenge"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		// Empty body is fine - auto-consent mode.
		JSON(w, http.StatusOK, map[string]any{"accepted": true})
		return
	}

	if req.ConsentChallenge == "" || h.cache == nil {
		JSON(w, http.StatusOK, map[string]any{"accepted": true})
		return
	}

	// Look up the challenge.
	data, err := h.cache.Get(r.Context(), "consent:"+req.ConsentChallenge)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_challenge", "consent challenge expired or invalid")
		return
	}
	_ = h.cache.Delete(r.Context(), "consent:"+req.ConsentChallenge)

	var challenge consentChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		Error(w, http.StatusInternalServerError, "internal", "failed to unmarshal challenge")
		return
	}

	// Redirect back to the authorize endpoint to complete the flow.
	JSON(w, http.StatusOK, map[string]any{
		"accepted":     true,
		"redirect_uri": challenge.RequestURL,
	})
}

// ConsentReject handles POST /api/v1/consent/reject.
func (h *OIDCHandler) ConsentReject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConsentChallenge string `json:"consent_challenge"`
	}
	_ = DecodeJSON(r, &req)

	if req.ConsentChallenge != "" && h.cache != nil {
		data, err := h.cache.Get(r.Context(), "consent:"+req.ConsentChallenge)
		if err == nil {
			_ = h.cache.Delete(r.Context(), "consent:"+req.ConsentChallenge)
			var challenge consentChallenge
			if json.Unmarshal(data, &challenge) == nil {
				JSON(w, http.StatusOK, map[string]any{
					"rejected":     true,
					"redirect_uri": challenge.RedirectURI + "?error=access_denied&error_description=user+denied+consent",
				})
				return
			}
		}
	}

	JSON(w, http.StatusOK, map[string]any{"rejected": true})
}

// storeConsentChallenge saves the authorize request so we can resume after consent.
func (h *OIDCHandler) storeConsentChallenge(r *http.Request, challenge *consentChallenge) (string, error) {
	token, err := generateRandomToken(24)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(challenge)
	if err := h.cache.Set(r.Context(), "consent:"+token, data, 10*time.Minute); err != nil {
		return "", err
	}
	return token, nil
}

// generateRandomToken is imported from the service package; reuse the helper here.
func generateRandomToken(length int) (string, error) {
	return service.GenerateRandomTokenExported(length)
}

