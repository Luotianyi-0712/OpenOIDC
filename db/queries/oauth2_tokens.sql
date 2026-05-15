-- ===========================================================================
-- Authorization Codes
-- ===========================================================================

-- name: CreateAuthorizationCode :exec
INSERT INTO oauth2_authorization_codes (
    signature, request_id, requested_at, client_id, scopes, granted_scopes,
    form_data, session_data, subject, active, requested_audience, granted_audience, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetAuthorizationCode :one
SELECT * FROM oauth2_authorization_codes
WHERE signature = $1
LIMIT 1;

-- name: DeleteAuthorizationCode :exec
DELETE FROM oauth2_authorization_codes
WHERE signature = $1;

-- name: InvalidateAuthorizationCode :exec
UPDATE oauth2_authorization_codes
SET active = FALSE
WHERE signature = $1;

-- name: DeleteAuthorizationCodesByRequestID :exec
DELETE FROM oauth2_authorization_codes
WHERE request_id = $1;

-- ===========================================================================
-- Access Tokens
-- ===========================================================================

-- name: CreateAccessToken :exec
INSERT INTO oauth2_access_tokens (
    signature, request_id, requested_at, client_id, scopes, granted_scopes,
    form_data, session_data, subject, active, requested_audience, granted_audience, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetAccessToken :one
SELECT * FROM oauth2_access_tokens
WHERE signature = $1
LIMIT 1;

-- name: DeleteAccessToken :exec
DELETE FROM oauth2_access_tokens
WHERE signature = $1;

-- name: RevokeAccessTokenByRequestID :exec
UPDATE oauth2_access_tokens
SET active = FALSE
WHERE request_id = $1;

-- name: DeleteAccessTokensByRequestID :exec
DELETE FROM oauth2_access_tokens
WHERE request_id = $1;

-- ===========================================================================
-- Refresh Tokens
-- ===========================================================================

-- name: CreateRefreshToken :exec
INSERT INTO oauth2_refresh_tokens (
    signature, request_id, requested_at, client_id, scopes, granted_scopes,
    form_data, session_data, subject, active, requested_audience, granted_audience, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetRefreshToken :one
SELECT * FROM oauth2_refresh_tokens
WHERE signature = $1
LIMIT 1;

-- name: DeleteRefreshToken :exec
DELETE FROM oauth2_refresh_tokens
WHERE signature = $1;

-- name: RevokeRefreshTokenByRequestID :exec
UPDATE oauth2_refresh_tokens
SET active = FALSE
WHERE request_id = $1;

-- name: DeleteRefreshTokensByRequestID :exec
DELETE FROM oauth2_refresh_tokens
WHERE request_id = $1;

-- ===========================================================================
-- OIDC Sessions
-- ===========================================================================

-- name: CreateOIDCSession :exec
INSERT INTO oauth2_oidc_sessions (
    signature, request_id, requested_at, client_id, scopes, granted_scopes,
    form_data, session_data, subject, active, requested_audience, granted_audience, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetOIDCSession :one
SELECT * FROM oauth2_oidc_sessions
WHERE signature = $1
LIMIT 1;

-- name: DeleteOIDCSession :exec
DELETE FROM oauth2_oidc_sessions
WHERE signature = $1;

-- ===========================================================================
-- PKCE Requests
-- ===========================================================================

-- name: CreatePKCERequest :exec
INSERT INTO oauth2_pkce_requests (
    signature, request_id, requested_at, client_id, scopes, granted_scopes,
    form_data, session_data, subject, active, requested_audience, granted_audience, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetPKCERequest :one
SELECT * FROM oauth2_pkce_requests
WHERE signature = $1
LIMIT 1;

-- name: DeletePKCERequest :exec
DELETE FROM oauth2_pkce_requests
WHERE signature = $1;
