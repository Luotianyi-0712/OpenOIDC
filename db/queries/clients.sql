-- name: CreateClient :one
INSERT INTO oidc_clients (
    client_id,
    client_secret_hash,
    name,
    description,
    logo_url,
    homepage_url,
    redirect_uris,
    post_logout_redirect_uris,
    grant_types,
    response_types,
    scopes,
    audience,
    token_endpoint_auth_method,
    protocol_type,
    min_security_level,
    require_pkce,
    require_consent,
    is_public,
    is_first_party,
    is_active,
    owner_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
)
RETURNING *;

-- name: GetClientByID :one
SELECT * FROM oidc_clients
WHERE id = $1
LIMIT 1;

-- name: GetClientByClientID :one
SELECT * FROM oidc_clients
WHERE client_id = $1
LIMIT 1;

-- name: ListClients :many
SELECT * FROM oidc_clients
WHERE (sqlc.narg('search')::text IS NULL
       OR name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR client_id ILIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: CountClients :one
SELECT COUNT(*)::bigint AS count
FROM oidc_clients
WHERE (sqlc.narg('search')::text IS NULL
       OR name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR client_id ILIKE '%' || sqlc.narg('search')::text || '%');

-- name: UpdateClient :exec
UPDATE oidc_clients
SET name = $2,
    description = $3,
    logo_url = $4,
    homepage_url = $5,
    redirect_uris = $6,
    post_logout_redirect_uris = $7,
    grant_types = $8,
    response_types = $9,
    scopes = $10,
    audience = $11,
    token_endpoint_auth_method = $12,
    protocol_type = $13,
    min_security_level = $14,
    require_pkce = $15,
    require_consent = $16,
    is_public = $17,
    is_first_party = $18,
    is_active = $19,
    metadata = $20,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteClient :exec
DELETE FROM oidc_clients
WHERE id = $1;

-- name: UpdateClientSecret :exec
UPDATE oidc_clients
SET client_secret_hash = $2,
    updated_at = NOW()
WHERE id = $1;
