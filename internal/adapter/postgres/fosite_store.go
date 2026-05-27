package postgres

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/anthropic/oidc-platform/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/fosite"
)

type FositeStore struct {
	db           *pgxpool.Pool
	secretCipher *service.SecretCipher
}

func NewFositeStore(db *pgxpool.Pool, secretCipher *service.SecretCipher) *FositeStore {
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
	audience      []string
}

func (c *fositeClient) GetID() string                      { return c.id }
func (c *fositeClient) GetHashedSecret() []byte            { return []byte(c.secret) }
func (c *fositeClient) GetRedirectURIs() []string          { return c.redirectURIs }
func (c *fositeClient) GetGrantTypes() fosite.Arguments    { return fosite.Arguments(c.grantTypes) }
func (c *fositeClient) GetResponseTypes() fosite.Arguments { return fosite.Arguments(c.responseTypes) }
func (c *fositeClient) GetScopes() fosite.Arguments        { return fosite.Arguments(c.scopes) }
func (c *fositeClient) IsPublic() bool                     { return c.public }
func (c *fositeClient) GetAudience() fosite.Arguments      { return fosite.Arguments(c.audience) }

// ---------------------------------------------------------------------------
// ClientManager
// ---------------------------------------------------------------------------

func (s *FositeStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	var (
		clientID, secret, authMethod string
		redirectURIs, grantTypes     []string
		responseTypes, scopes        []string
		audience                     []string
		isPublic                     bool
	)
	err := s.db.QueryRow(ctx,
		`SELECT client_id, client_secret_encrypted, redirect_uris, grant_types, response_types,
		 scopes, audience, token_endpoint_auth_method, is_public
		 FROM oidc_clients WHERE client_id = $1 AND is_active = TRUE`,
		id,
	).Scan(&clientID, &secret, &redirectURIs, &grantTypes, &responseTypes, &scopes, &audience, &authMethod, &isPublic)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fosite.ErrNotFound
		}
		return nil, err
	}
	plainSecret, err := s.secretCipher.Decrypt(secret)
	if err != nil {
		return nil, err
	}
	return &fositeClient{
		id:            clientID,
		secret:        plainSecret,
		redirectURIs:  redirectURIs,
		grantTypes:    grantTypes,
		responseTypes: responseTypes,
		scopes:        scopes,
		public:        isPublic,
		audience:      audience,
	}, nil
}

func (s *FositeStore) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	return nil
}

func (s *FositeStore) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error {
	return nil
}

// ---------------------------------------------------------------------------
// Generic session storage helpers
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

func encodeRequest(req fosite.Requester) (*sessionEnvelope, []byte, error) {
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
			return nil, nil, fmt.Errorf("encode session: %w", err)
		}
		env.Session = sessData
	}

	// Still use gob for the envelope
	envBuf := bytes.Buffer{}
	if err := gob.NewEncoder(&envBuf).Encode(env); err != nil {
		return nil, nil, fmt.Errorf("encode envelope: %w", err)
	}
	return env, envBuf.Bytes(), nil
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

func (s *FositeStore) saveSession(ctx context.Context, table, signature string, req fosite.Requester) error {
	_, data, err := encodeRequest(req)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(
		`INSERT INTO %s (signature, request_id, requested_at, client_id, scopes, granted_scopes,
		 session_data, subject, active, requested_audience, granted_audience, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, $9, $10, $11)
		 ON CONFLICT (signature) DO UPDATE SET session_data = EXCLUDED.session_data, active = TRUE`,
		table,
	)
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
	_, err = s.db.Exec(ctx, query,
		signature,
		req.GetID(),
		req.GetRequestedAt(),
		req.GetClient().GetID(),
		strings.Join(req.GetRequestedScopes(), " "),
		strings.Join(req.GetGrantedScopes(), " "),
		data,
		subject,
		strings.Join(req.GetRequestedAudience(), " "),
		strings.Join(req.GetGrantedAudience(), " "),
		toNullTime(expiresAt),
	)
	return err
}

func (s *FositeStore) loadSession(ctx context.Context, table, signature string, session fosite.Session) (fosite.Requester, error) {
	var data []byte
	var active bool
	err := s.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT session_data, active FROM %s WHERE signature = $1`, table),
		signature,
	).Scan(&data, &active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fosite.ErrNotFound
		}
		return nil, err
	}
	req, err := decodeRequest(data, session)
	if err != nil {
		return nil, err
	}
	if !active {
		return req, fosite.ErrInactiveToken
	}
	return req, nil
}

func (s *FositeStore) deactivateSession(ctx context.Context, table, signature string) error {
	_, err := s.db.Exec(ctx,
		fmt.Sprintf(`UPDATE %s SET active = FALSE WHERE signature = $1`, table),
		signature,
	)
	return err
}

func (s *FositeStore) deleteSession(ctx context.Context, table, signature string) error {
	_, err := s.db.Exec(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE signature = $1`, table),
		signature,
	)
	return err
}

func (s *FositeStore) revokeByRequestID(ctx context.Context, table, requestID string) error {
	_, err := s.db.Exec(ctx,
		fmt.Sprintf(`UPDATE %s SET active = FALSE WHERE request_id = $1`, table),
		requestID,
	)
	return err
}

// ---------------------------------------------------------------------------
// AuthorizeCodeStorage
// ---------------------------------------------------------------------------

func (s *FositeStore) CreateAuthorizeCodeSession(ctx context.Context, code string, req fosite.Requester) error {
	return s.saveSession(ctx, "oauth2_authorization_codes", code, req)
}

func (s *FositeStore) GetAuthorizeCodeSession(ctx context.Context, code string, session fosite.Session) (fosite.Requester, error) {
	return s.loadSession(ctx, "oauth2_authorization_codes", code, session)
}

func (s *FositeStore) InvalidateAuthorizeCodeSession(ctx context.Context, code string) error {
	return s.deactivateSession(ctx, "oauth2_authorization_codes", code)
}

// ---------------------------------------------------------------------------
// AccessTokenStorage
// ---------------------------------------------------------------------------

func (s *FositeStore) CreateAccessTokenSession(ctx context.Context, signature string, req fosite.Requester) error {
	return s.saveSession(ctx, "oauth2_access_tokens", signature, req)
}

func (s *FositeStore) GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.loadSession(ctx, "oauth2_access_tokens", signature, session)
}

func (s *FositeStore) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, "oauth2_access_tokens", signature)
}

func (s *FositeStore) RevokeAccessToken(ctx context.Context, requestID string) error {
	return s.revokeByRequestID(ctx, "oauth2_access_tokens", requestID)
}

// ---------------------------------------------------------------------------
// RefreshTokenStorage
// ---------------------------------------------------------------------------

func (s *FositeStore) CreateRefreshTokenSession(ctx context.Context, signature string, accessSignature string, req fosite.Requester) error {
	return s.saveSession(ctx, "oauth2_refresh_tokens", signature, req)
}

func (s *FositeStore) GetRefreshTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.loadSession(ctx, "oauth2_refresh_tokens", signature, session)
}

func (s *FositeStore) DeleteRefreshTokenSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, "oauth2_refresh_tokens", signature)
}

func (s *FositeStore) RevokeRefreshToken(ctx context.Context, requestID string) error {
	return s.revokeByRequestID(ctx, "oauth2_refresh_tokens", requestID)
}

func (s *FositeStore) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error {
	return s.revokeByRequestID(ctx, "oauth2_refresh_tokens", requestID)
}

// ---------------------------------------------------------------------------
// OpenIDConnectRequestStorage
// ---------------------------------------------------------------------------

func (s *FositeStore) CreateOpenIDConnectSession(ctx context.Context, authorizeCode string, req fosite.Requester) error {
	return s.saveSession(ctx, "oauth2_oidc_sessions", authorizeCode, req)
}

func (s *FositeStore) GetOpenIDConnectSession(ctx context.Context, authorizeCode string, req fosite.Requester) (fosite.Requester, error) {
	return s.loadSession(ctx, "oauth2_oidc_sessions", authorizeCode, req.GetSession())
}

func (s *FositeStore) DeleteOpenIDConnectSession(ctx context.Context, authorizeCode string) error {
	return s.deleteSession(ctx, "oauth2_oidc_sessions", authorizeCode)
}

// ---------------------------------------------------------------------------
// PKCERequestStorage
// ---------------------------------------------------------------------------

func (s *FositeStore) CreatePKCERequestSession(ctx context.Context, signature string, req fosite.Requester) error {
	return s.saveSession(ctx, "oauth2_pkce_requests", signature, req)
}

func (s *FositeStore) GetPKCERequestSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.loadSession(ctx, "oauth2_pkce_requests", signature, session)
}

func (s *FositeStore) DeletePKCERequestSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, "oauth2_pkce_requests", signature)
}
