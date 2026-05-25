package domain

import (
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type PasskeyCredential struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	CredentialID    []byte     `json:"-"`
	PublicKey       []byte     `json:"-"`
	AttestationType string     `json:"attestation_type"`
	Transport       []string   `json:"transport"`
	SignCount       uint32     `json:"sign_count"`
	AAGUID         []byte     `json:"-"`
	Name            string     `json:"name"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// WebAuthnUser wraps a domain.User with its passkey credentials to satisfy
// the webauthn.User interface required by go-webauthn.
type WebAuthnUser struct {
	User        *User
	Credentials []*PasskeyCredential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	b, _ := u.User.ID.MarshalBinary()
	return b
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.User.Email
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	if u.User.DisplayName != "" {
		return u.User.DisplayName
	}
	return u.User.Email
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, 0, len(u.Credentials))
	for _, c := range u.Credentials {
		transport := make([]protocol.AuthenticatorTransport, 0, len(c.Transport))
		for _, t := range c.Transport {
			transport = append(transport, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transport,
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AAGUID,
				SignCount:  c.SignCount,
			},
		})
	}
	return creds
}

var _ webauthn.User = (*WebAuthnUser)(nil)