package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ory/fosite"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/domain"
	mw "github.com/anthropic/oidc-platform/internal/handler/middleware"
	"github.com/anthropic/oidc-platform/internal/oidcprovider"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/anthropic/oidc-platform/internal/service"
)

type OIDCHandler struct {
	provider     fosite.OAuth2Provider
	userRepo     port.UserRepository
	clientSvc    *service.ClientService
	accessCtrl   *service.AccessControlService
	sessionSvc   *service.SessionService
	settingsRepo port.SettingsRepository
	cache        port.Cache
	serverCfg    config.ServerConfig
	loginURL     string
}

func NewOIDCHandler(
	provider fosite.OAuth2Provider,
	userRepo port.UserRepository,
	clientSvc *service.ClientService,
	accessCtrl *service.AccessControlService,
	sessionSvc *service.SessionService,
	settingsRepo port.SettingsRepository,
	serverCfg config.ServerConfig,
	loginURL string,
) *OIDCHandler {
	return &OIDCHandler{
		provider:     provider,
		userRepo:     userRepo,
		clientSvc:    clientSvc,
		accessCtrl:   accessCtrl,
		sessionSvc:   sessionSvc,
		settingsRepo: settingsRepo,
		serverCfg:    serverCfg,
		loginURL:     loginURL,
	}
}

// SetCache sets the cache dependency for storing consent challenges.
func (h *OIDCHandler) SetCache(cache port.Cache) {
	h.cache = cache
}

// consentChallenge stores the OIDC authorize request info so we can resume after user consent.
type consentChallenge struct {
	UserID            string   `json:"user_id"`
	ClientID          string   `json:"client_id"`
	ClientName        string   `json:"client_name"`
	ClientDescription string   `json:"client_description"`
	ClientLogoURL     string   `json:"client_logo_url"`
	DeveloperID       string   `json:"developer_id"`
	WebsiteURL        string   `json:"website_url"`
	Scopes            []string `json:"scopes"`
	RedirectURI       string   `json:"redirect_uri"`
	RequestURL        string   `json:"request_url"`
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
	if user.Status != domain.UserStatusActive {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrAccessDenied.WithHint("user account is not active"))
		return
	}

	client, err := h.clientSvc.GetClientByClientID(ctx, ar.GetClient().GetID())
	if err != nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrInvalidClient.WithWrap(err))
		return
	}

	// Check if the client is disabled
	if !client.IsActive {
		// Redirect to a friendly error page instead of returning JSON error
		errorURL := "/error?type=app_disabled&app=" + url.QueryEscape(client.ClientName)
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}

	if user.SecurityLevel < client.MinSecurityLevel {
		h.provider.WriteAuthorizeError(ctx, w, ar,
			fosite.ErrAccessDenied.WithHint("security level insufficient for this client"))
		return
	}
	if client.RequireEmailVerified && !user.EmailVerified {
		h.provider.WriteAuthorizeError(ctx, w, ar,
			fosite.ErrAccessDenied.WithHint("email verification required for this client"))
		return
	}

	if ok, reason := h.accessCtrl.CheckAccess(ctx, client, user, clientIP(r)); !ok {
		h.provider.WriteAuthorizeError(ctx, w, ar,
			fosite.ErrAccessDenied.WithHint(reason))
		return
	}

	consentToken := r.URL.Query().Get("consent_challenge")
	if consentToken == "" {
		if h.cache == nil {
			h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrServerError.WithHint("consent cache unavailable"))
			return
		}
		developerID := "platform"
		if client.OwnerUserID != nil {
			developerID = client.OwnerUserID.String()
		}
		redirectURI := ar.GetRedirectURI().String()
		challenge := &consentChallenge{
			UserID:            user.ID.String(),
			ClientID:          client.ClientID,
			ClientName:        client.ClientName,
			ClientDescription: client.Description,
			ClientLogoURL:     client.LogoURL,
			DeveloperID:       developerID,
			WebsiteURL:        client.HomepageURL,
			Scopes:            ar.GetRequestedScopes(),
			RedirectURI:       redirectURI,
			RequestURL:        r.URL.RequestURI(),
		}
		token, err := h.storeConsentChallenge(r, challenge)
		if err != nil {
			h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrServerError.WithWrap(err))
			return
		}
		consentURL := "/authorize?" + url.Values{
			"consent_challenge": {token},
		}.Encode()
		http.Redirect(w, r, consentURL, http.StatusFound)
		return
	}
	if h.cache == nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrServerError.WithHint("consent cache unavailable"))
		return
	}
	acceptedData, err := h.cache.Get(ctx, "consent_accepted:"+consentToken)
	if err != nil {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrAccessDenied.WithHint("consent challenge not accepted"))
		return
	}
	_ = h.cache.Delete(ctx, "consent_accepted:"+consentToken)
	var accepted consentChallenge
	if err := json.Unmarshal(acceptedData, &accepted); err != nil || !consentChallengeMatches(accepted, user.ID.String(), client.ClientID, ar.GetRedirectURI().String(), ar.GetRequestedScopes()) {
		h.provider.WriteAuthorizeError(ctx, w, ar, fosite.ErrAccessDenied.WithHint("consent challenge does not match request"))
		return
	}

	// User accepted consent: grant all requested scopes and issue response.
	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}
	for _, aud := range ar.GetRequestedAudience() {
		ar.GrantAudience(aud)
	}

	session := oidcprovider.NewSession(user.ID.String(), user.SecurityLevel, h.publicIssuer(r), client.ClientID, user.ID.String())
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

func (h *OIDCHandler) publicIssuer(r *http.Request) string {
	if h.settingsRepo != nil {
		setting, err := h.settingsRepo.Get(r.Context(), "site_url")
		if err == nil && setting != nil {
			if value := strings.TrimRight(strings.TrimSpace(setting.Value), "/"); value != "" {
				return value
			}
		}
	}
	if value := strings.TrimRight(strings.TrimSpace(h.serverCfg.Issuer), "/"); value != "" {
		return value
	}
	return strings.TrimRight(strings.TrimSpace(h.serverCfg.BaseURL), "/")
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

	// Check if the client is disabled
	client, err := h.clientSvc.GetClientByClientID(ctx, accessReq.GetClient().GetID())
	if err == nil && !client.IsActive {
		h.provider.WriteAccessError(ctx, w, accessReq, fosite.ErrInvalidClient.WithHint("client is disabled"))
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

	// Check if the client is disabled - if so, return inactive token
	if resp.IsActive() {
		clientID := resp.GetAccessRequester().GetClient().GetID()
		client, err := h.clientSvc.GetClientByClientID(ctx, clientID)
		if err == nil && !client.IsActive {
			// Return an inactive response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"active":false}`))
			return
		}
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

	// Check if the client is disabled
	client, err := h.clientSvc.GetClientByClientID(ctx, ar.GetClient().GetID())
	if err == nil && !client.IsActive {
		Error(w, http.StatusUnauthorized, "invalid_client", "client is disabled")
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

// ConsentContext handles GET /api/v1/consent/context.
func (h *OIDCHandler) ConsentContext(w http.ResponseWriter, r *http.Request) {
	challengeID := r.URL.Query().Get("consent_challenge")
	if challengeID == "" || h.cache == nil {
		Error(w, http.StatusBadRequest, "invalid_challenge", "consent challenge required")
		return
	}
	data, err := h.cache.Get(r.Context(), "consent:"+challengeID)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_challenge", "consent challenge expired or invalid")
		return
	}
	var challenge consentChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		Error(w, http.StatusInternalServerError, "internal", "failed to unmarshal challenge")
		return
	}
	userID, err := mw.GetUserID(r.Context())
	if err != nil || challenge.UserID != userID.String() {
		Error(w, http.StatusForbidden, "forbidden", "consent challenge does not belong to current user")
		return
	}
	developerName := "platform"
	if developerID, err := uuid.Parse(challenge.DeveloperID); err == nil {
		if developer, err := h.userRepo.GetByID(r.Context(), developerID); err == nil {
			developerName = developer.DisplayName
			if developerName == "" {
				developerName = developer.Email
			}
		}
	}
	JSON(w, http.StatusOK, map[string]any{
		"client": map[string]any{
			"client_id":    challenge.ClientID,
			"name":         challenge.ClientName,
			"description":  challenge.ClientDescription,
			"logo_url":     challenge.ClientLogoURL,
			"homepage_url": challenge.WebsiteURL,
		},
		"developer": map[string]any{
			"name": developerName,
		},
		"scopes": challenge.Scopes,
	})
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
	var challenge consentChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		Error(w, http.StatusInternalServerError, "internal", "failed to unmarshal challenge")
		return
	}
	userID, err := mw.GetUserID(r.Context())
	if err != nil || challenge.UserID != userID.String() {
		Error(w, http.StatusForbidden, "forbidden", "consent challenge does not belong to current user")
		return
	}
	_ = h.cache.Delete(r.Context(), "consent:"+req.ConsentChallenge)
	if err := h.cache.Set(r.Context(), "consent_accepted:"+req.ConsentChallenge, data, 5*time.Minute); err != nil {
		Error(w, http.StatusInternalServerError, "internal", "failed to accept challenge")
		return
	}

	// Redirect back to the authorize endpoint to complete the flow.
	resumeURL, err := url.Parse(challenge.RequestURL)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", "failed to parse challenge request")
		return
	}
	q := resumeURL.Query()
	q.Set("consent_challenge", req.ConsentChallenge)
	resumeURL.RawQuery = q.Encode()
	JSON(w, http.StatusOK, map[string]any{
		"accepted":     true,
		"redirect_uri": resumeURL.String(),
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
			var challenge consentChallenge
			if err := json.Unmarshal(data, &challenge); err != nil {
				Error(w, http.StatusInternalServerError, "internal", "failed to unmarshal challenge")
				return
			}
			userID, err := mw.GetUserID(r.Context())
			if err != nil || challenge.UserID != userID.String() {
				Error(w, http.StatusForbidden, "forbidden", "consent challenge does not belong to current user")
				return
			}
			_ = h.cache.Delete(r.Context(), "consent:"+req.ConsentChallenge)
			JSON(w, http.StatusOK, map[string]any{
				"rejected":     true,
				"redirect_uri": buildConsentRejectRedirect(challenge),
			})
			return
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

func consentChallengeMatches(challenge consentChallenge, userID, clientID, redirectURI string, scopes []string) bool {
	if challenge.UserID != userID || challenge.ClientID != clientID || challenge.RedirectURI != redirectURI {
		return false
	}
	if len(challenge.Scopes) != len(scopes) {
		return false
	}
	seen := make(map[string]int, len(challenge.Scopes))
	for _, scope := range challenge.Scopes {
		seen[scope]++
	}
	for _, scope := range scopes {
		if seen[scope] == 0 {
			return false
		}
		seen[scope]--
	}
	return true
}

func buildConsentRejectRedirect(challenge consentChallenge) string {
	redirectURL, err := url.Parse(challenge.RedirectURI)
	if err != nil {
		return challenge.RedirectURI
	}
	q := redirectURL.Query()
	q.Set("error", "access_denied")
	q.Set("error_description", "user denied consent")
	if requestURL, err := url.Parse(challenge.RequestURL); err == nil {
		if state := requestURL.Query().Get("state"); state != "" {
			q.Set("state", state)
		}
	}
	redirectURL.RawQuery = q.Encode()
	return redirectURL.String()
}
