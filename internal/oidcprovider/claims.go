package oidcprovider

// AddCustomClaims merges custom claims (e.g. security_level, user_id) into the session's ID token claims.
// fosite reads claims from the session passed to NewAuthorizeRequest/NewAccessRequest, so we ensure
// the Extra map is populated.
func AddCustomClaims(session *Session, claims map[string]any) {
	if session == nil || session.DefaultSession == nil || session.DefaultSession.Claims == nil {
		return
	}
	if session.DefaultSession.Claims.Extra == nil {
		session.DefaultSession.Claims.Extra = map[string]interface{}{}
	}
	for k, v := range claims {
		session.DefaultSession.Claims.Extra[k] = v
	}
}
