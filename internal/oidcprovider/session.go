package oidcprovider

import (
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"
)

type Session struct {
	*openid.DefaultSession
	SecurityLevel int    `json:"security_level"`
	UserID        string `json:"user_id"`
}

func NewSession(userID string, securityLevel int, issuer, clientID, subject string) *Session {
	now := time.Now().UTC()
	return &Session{
		DefaultSession: &openid.DefaultSession{
			Claims: &jwt.IDTokenClaims{
				Subject:   subject,
				Issuer:    issuer,
				Audience:  []string{clientID},
				IssuedAt:  now,
				ExpiresAt: now.Add(time.Hour),
				Extra: map[string]interface{}{
					"security_level": securityLevel,
					"user_id":        userID,
				},
			},
			Headers: &jwt.Headers{Extra: map[string]interface{}{}},
			Subject: subject,
		},
		SecurityLevel: securityLevel,
		UserID:        userID,
	}
}

func (s *Session) Clone() fosite.Session {
	if s == nil {
		return nil
	}
	cloned := *s
	if s.DefaultSession != nil {
		ds := *s.DefaultSession
		cloned.DefaultSession = &ds
	}
	return &cloned
}
