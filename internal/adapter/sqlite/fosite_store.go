package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/anthropic/oidc-platform/internal/service"
	"github.com/google/uuid"
	"github.com/ory/fosite"
)

// FositeStore implements fosite OAuth2 storage interfaces using SQLite.
// It covers: fosite.Storage, oauth2.CoreStorage, openid.OpenIDConnectRequestStorage,
// pkce.PKCERequestStorage, and fosite.ClientManager.
type FositeStore struct {
	db           *sql.DB
	secretCipher *service.SecretCipher
}

// NewFositeStore returns a new FositeStore.
func NewFositeStore(db *sql.DB, secretCipher *service.SecretCipher) *FositeStore {
	return &FositeStore{db: db, secretCipher: secretCipher}
}

// ---------------------------------------------------------------------------
// fosite.Client implementation
// ---------------------------------------------------------------------------

type fositeClient struct {
	id            string
	secret        string
	redirectURIs  []string
	grantTypes    []string
	responseTypes []string
	scopes        []string
	public        bool
}

func (c *fositeClient) GetID() string                      { return c.id }
func (c *fositeClient) GetHashedSecret() []byte            { return []byte(c.secret) }
func (c *fositeClient) GetRedirectURIs() []string          { return c.redirectURIs }
func (c *fositeClient) GetGrantTypes() fosite.Arguments    { return fosite.Arguments(c.grantTypes) }
func (c *fositeClient) GetResponseTypes() fosite.Arguments { return fosite.Arguments(c.responseTypes) }
func (c *fositeClient) GetScopes() fosite.Arguments        { return fosite.Arguments(c.scopes) }
func (c *fositeClient) IsPublic() bool                     { return c.public }
func (c *fositeClient) GetAudience() fosite.Arguments      { return nil }

// ---------------------------------------------------------------------------
// ClientManager / fosite.Storage
// ---------------------------------------------------------------------------

// GetClient retrieves an OIDC client by client_id from the oidc_clients table.
func (s *FositeStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	var (
		clientID, secret                 string
		redirectURIsJSON, grantTypesJSON string
		responseTypesJSON, scopesJSON    string
		isConfidential                   bool
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT client_id, client_secret_encrypted, redirect_uris, grant_types, response_types,
		 scopes, is_confidential
		 FROM oidc_clients WHERE client_id = ? AND is_active = 1`,
		id,
	).Scan(&clientID, &secret, &redirectURIsJSON, &grantTypesJSON,
		&responseTypesJSON, &scopesJSON, &isConfidential)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fosite.ErrNotFound
		}
		return nil, err
	}
	plainSecret, err := s.secretCipher.Decrypt(secret)
	if err != nil {
		return nil, err
	}

	var redirectURIs, grantTypes, responseTypes, scopes []string
	_ = json.Unmarshal([]byte(redirectURIsJSON), &redirectURIs)
	_ = json.Unmarshal([]byte(grantTypesJSON), &grantTypes)
	_ = json.Unmarshal([]byte(responseTypesJSON), &responseTypes)
	_ = json.Unmarshal([]byte(scopesJSON), &scopes)

	return &fositeClient{
		id:            clientID,
		secret:        plainSecret,
		redirectURIs:  redirectURIs,
		grantTypes:    grantTypes,
		responseTypes: responseTypes,
		scopes:        scopes,
		public:        !isConfidential,
	}, nil
}

// ClientAssertionJWTValid is a no-op for this implementation.
func (s *FositeStore) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	return nil
}

// SetClientAssertionJWT is a no-op for this implementation.
func (s *FositeStore) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error {
	return nil
}

// ---------------------------------------------------------------------------
// Gob encoding helpers
// ---------------------------------------------------------------------------

type sessionEnvelope struct {
	RequestID         string
	RequestedAt       time.Time
	ClientID          string
	Scopes            []string
	GrantedScopes     []string
	Form              map[string][]string
	Session           []byte
	RequestedAudience []string
	GrantedAudience   []string
}

func encodeRequest(req fosite.Requester) ([]byte, error) {
	env := &sessionEnvelope{
		RequestID:         req.GetID(),
		RequestedAt:       req.GetRequestedAt(),
		ClientID:          req.GetClient().GetID(),
		Scopes:            req.GetRequestedScopes(),
		GrantedScopes:     req.GetGrantedScopes(),
		Form:              req.GetRequestForm(),
		RequestedAudience: req.GetRequestedAudience(),
		GrantedAudience:   req.GetGrantedAudience(),
	}

	// Use JSON instead of gob to handle nil pointers in session
	if sess := req.GetSession(); sess != nil {
		sessData, err := json.Marshal(sess)
		if err != nil {
			return nil, fmt.Errorf("encode session: %w", err)
		}
		env.Session = sessData
	}

	// Still use gob for the envelope (which doesn't have nil pointer issues)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(env); err != nil {
		return nil, fmt.Errorf("encode envelope: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeRequest(data []byte, session fosite.Session) (fosite.Requester, error) {
	var env sessionEnvelope
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}

	// Use JSON to decode session (matching the encoding)
	if len(env.Session) > 0 && session != nil {
		if err := json.Unmarshal(env.Session, session); err != nil {
			return nil, fmt.Errorf("decode session: %w", err)
		}
	}

	req := &fosite.Request{
		ID:                env.RequestID,
		RequestedAt:       env.RequestedAt,
		Client:            &fositeClient{id: env.ClientID},
		RequestedScope:    fosite.Arguments(env.Scopes),
		GrantedScope:      fosite.Arguments(env.GrantedScopes),
		Form:              url.Values(env.Form),
		Session:           session,
		RequestedAudience: fosite.Arguments(env.RequestedAudience),
		GrantedAudience:   fosite.Arguments(env.GrantedAudience),
	}
	return req, nil
}

// ---------------------------------------------------------------------------
// Generic session storage helpers
// ---------------------------------------------------------------------------

func (s *FositeStore) saveSession(ctx context.Context, sessionType, signature string, req fosite.Requester) error {
	data, err := encodeRequest(req)
	if err != nil {
		return err
	}

	subject := ""
	if sess := req.GetSession(); sess != nil {
		if u, ok := sess.(interface{ GetSubject() string }); ok {
			subject = u.GetSubject()
		}
	}

	var expiresAt *time.Time
	if sess := req.GetSession(); sess != nil {
		if e, ok := sess.(interface {
			GetExpiresAt(fosite.TokenType) time.Time
		}); ok {
			t := e.GetExpiresAt(fosite.AccessToken)
			if !t.IsZero() {
				expiresAt = &t
			}
		}
	}

	id := uuid.New().String()
	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO oauth2_sessions
		 (id, request_id, session_type, client_id, signature, subject, data, created_at, expires_at, active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		id,
		req.GetID(),
		sessionType,
		req.GetClient().GetID(),
		signature,
		subject,
		data,
		time.Now().UTC(),
		toNullTime(expiresAt),
	)
	if err != nil {
		return fmt.Errorf("failed to save %s session: %w", sessionType, err)
	}
	return nil
}

func (s *FositeStore) loadSession(ctx context.Context, sessionType, signature string, session fosite.Session) (fosite.Requester, bool, error) {
	var data []byte
	var active bool
	err := s.db.QueryRowContext(ctx,
		`SELECT data, active FROM oauth2_sessions WHERE signature = ? AND session_type = ?`,
		signature, sessionType,
	).Scan(&data, &active)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, fosite.ErrNotFound
		}
		return nil, false, err
	}
	req, err := decodeRequest(data, session)
	if err != nil {
		return nil, false, err
	}
	return req, active, nil
}

func (s *FositeStore) deactivateSession(ctx context.Context, sessionType, signature string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE oauth2_sessions SET active = 0 WHERE signature = ? AND session_type = ?`,
		signature, sessionType,
	)
	return err
}

func (s *FositeStore) deleteSession(ctx context.Context, sessionType, signature string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM oauth2_sessions WHERE signature = ? AND session_type = ?`,
		signature, sessionType,
	)
	return err
}

func (s *FositeStore) revokeByRequestID(ctx context.Context, sessionType, requestID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE oauth2_sessions SET active = 0 WHERE request_id = ? AND session_type = ?`,
		requestID, sessionType,
	)
	return err
}

// ---------------------------------------------------------------------------
// AuthorizeCodeStorage
// ---------------------------------------------------------------------------

// CreateAuthorizeCodeSession stores an authorization code session.
func (s *FositeStore) CreateAuthorizeCodeSession(ctx context.Context, code string, req fosite.Requester) error {
	return s.saveSession(ctx, "auth_code", code, req)
}

// GetAuthorizeCodeSession retrieves an authorization code session.
// Returns fosite.ErrInvalidatedAuthorizeCode if the session exists but is inactive.
func (s *FositeStore) GetAuthorizeCodeSession(ctx context.Context, code string, session fosite.Session) (fosite.Requester, error) {
	req, active, err := s.loadSession(ctx, "auth_code", code, session)
	if err != nil {
		return nil, err
	}
	if !active {
		return req, fosite.ErrInvalidatedAuthorizeCode
	}
	return req, nil
}

// InvalidateAuthorizeCodeSession marks an authorization code session as inactive.
func (s *FositeStore) InvalidateAuthorizeCodeSession(ctx context.Context, code string) error {
	return s.deactivateSession(ctx, "auth_code", code)
}

// ---------------------------------------------------------------------------
// AccessTokenStorage
// ---------------------------------------------------------------------------

// CreateAccessTokenSession stores an access token session.
func (s *FositeStore) CreateAccessTokenSession(ctx context.Context, signature string, req fosite.Requester) error {
	return s.saveSession(ctx, "access_token", signature, req)
}

// GetAccessTokenSession retrieves an access token session.
func (s *FositeStore) GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	req, active, err := s.loadSession(ctx, "access_token", signature, session)
	if err != nil {
		return nil, err
	}
	if !active {
		return req, fosite.ErrInactiveToken
	}
	return req, nil
}

// DeleteAccessTokenSession removes an access token session.
func (s *FositeStore) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, "access_token", signature)
}

// RevokeAccessToken deactivates all access token sessions for a request ID.
func (s *FositeStore) RevokeAccessToken(ctx context.Context, requestID string) error {
	return s.revokeByRequestID(ctx, "access_token", requestID)
}

// ---------------------------------------------------------------------------
// RefreshTokenStorage
// ---------------------------------------------------------------------------

// CreateRefreshTokenSession stores a refresh token session.
func (s *FositeStore) CreateRefreshTokenSession(ctx context.Context, signature string, accessSignature string, req fosite.Requester) error {
	return s.saveSession(ctx, "refresh_token", signature, req)
}

// GetRefreshTokenSession retrieves a refresh token session.
// Returns fosite.ErrInactiveToken if the session exists but is inactive.
func (s *FositeStore) GetRefreshTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	req, active, err := s.loadSession(ctx, "refresh_token", signature, session)
	if err != nil {
		return nil, err
	}
	if !active {
		return req, fosite.ErrInactiveToken
	}
	return req, nil
}

// DeleteRefreshTokenSession removes a refresh token session.
func (s *FositeStore) DeleteRefreshTokenSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, "refresh_token", signature)
}

// RevokeRefreshToken deactivates all refresh token sessions for a request ID.
func (s *FositeStore) RevokeRefreshToken(ctx context.Context, requestID string) error {
	return s.revokeByRequestID(ctx, "refresh_token", requestID)
}

// RotateRefreshToken rotates a refresh token by deactivating the old one.
func (s *FositeStore) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error {
	return s.revokeByRequestID(ctx, "refresh_token", requestID)
}

// ---------------------------------------------------------------------------
// OpenIDConnectRequestStorage
// ---------------------------------------------------------------------------

// CreateOpenIDConnectSession stores an OpenID Connect session.
func (s *FositeStore) CreateOpenIDConnectSession(ctx context.Context, authorizeCode string, req fosite.Requester) error {
	return s.saveSession(ctx, "oidc", authorizeCode, req)
}

// GetOpenIDConnectSession retrieves an OpenID Connect session.
func (s *FositeStore) GetOpenIDConnectSession(ctx context.Context, authorizeCode string, req fosite.Requester) (fosite.Requester, error) {
	result, active, err := s.loadSession(ctx, "oidc", authorizeCode, req.GetSession())
	if err != nil {
		return nil, err
	}
	if !active {
		return result, fosite.ErrInactiveToken
	}
	return result, nil
}

// DeleteOpenIDConnectSession removes an OpenID Connect session.
func (s *FositeStore) DeleteOpenIDConnectSession(ctx context.Context, authorizeCode string) error {
	return s.deleteSession(ctx, "oidc", authorizeCode)
}

// ---------------------------------------------------------------------------
// PKCERequestStorage
// ---------------------------------------------------------------------------

// CreatePKCERequestSession stores a PKCE request session.
func (s *FositeStore) CreatePKCERequestSession(ctx context.Context, signature string, req fosite.Requester) error {
	return s.saveSession(ctx, "pkce", signature, req)
}

// GetPKCERequestSession retrieves a PKCE request session.
func (s *FositeStore) GetPKCERequestSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	req, active, err := s.loadSession(ctx, "pkce", signature, session)
	if err != nil {
		return nil, err
	}
	if !active {
		return req, fosite.ErrInactiveToken
	}
	return req, nil
}

// DeletePKCERequestSession removes a PKCE request session.
func (s *FositeStore) DeletePKCERequestSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, "pkce", signature)
}
