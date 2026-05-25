package handler

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"

	"github.com/anthropic/oidc-platform/internal/port"
)

type WellKnownHandler struct {
	baseURL        string
	signingKeyRepo port.SigningKeyRepository
}

func NewWellKnownHandler(baseURL string, signingKeyRepo port.SigningKeyRepository) *WellKnownHandler {
	return &WellKnownHandler{baseURL: baseURL, signingKeyRepo: signingKeyRepo}
}

func (h *WellKnownHandler) Discovery(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"issuer":                                h.baseURL,
		"authorization_endpoint":                h.baseURL + "/oauth2/authorize",
		"token_endpoint":                        h.baseURL + "/oauth2/token",
		"userinfo_endpoint":                     h.baseURL + "/oauth2/userinfo",
		"jwks_uri":                              h.baseURL + "/jwks.json",
		"revocation_endpoint":                   h.baseURL + "/oauth2/revoke",
		"introspection_endpoint":                h.baseURL + "/oauth2/introspect",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "security_level", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"claims_supported": []string{
			"sub", "iss", "aud", "exp", "iat",
			"email", "email_verified", "name", "avatar_url", "alias", "security_level",
		},
		"code_challenge_methods_supported": []string{"plain", "S256"},
	}
	Raw(w, http.StatusOK, doc)
}

func (h *WellKnownHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	keys, err := h.signingKeyRepo.List(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	jwks := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		pub, err := parsePublicKey(k.PublicKey)
		if err != nil {
			continue
		}
		jwks = append(jwks, map[string]any{
			"kty": "RSA",
			"use": "sig",
			"alg": k.Algorithm,
			"kid": k.KeyID,
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		})
	}
	Raw(w, http.StatusOK, map[string]any{"keys": jwks})
}

func parsePublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	if block.Type == "RSA PUBLIC KEY" {
		return x509.ParsePKCS1PublicKey(block.Bytes)
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return pub, nil
}
