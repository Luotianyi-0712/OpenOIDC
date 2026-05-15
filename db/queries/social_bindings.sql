-- name: CreateBinding :one
INSERT INTO social_bindings (
    user_id,
    provider,
    provider_uid,
    provider_email,
    provider_name,
    provider_avatar,
    access_token,
    refresh_token,
    token_expiry,
    raw_profile,
    verified_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetBindingByID :one
SELECT * FROM social_bindings
WHERE id = $1
LIMIT 1;

-- name: GetBindingByProviderUID :one
SELECT * FROM social_bindings
WHERE provider = $1 AND provider_uid = $2
LIMIT 1;

-- name: ListBindingsByUser :many
SELECT * FROM social_bindings
WHERE user_id = $1
ORDER BY bound_at DESC;

-- name: GetBindingByUserAndProvider :one
SELECT * FROM social_bindings
WHERE user_id = $1 AND provider = $2
LIMIT 1;

-- name: DeleteBinding :exec
DELETE FROM social_bindings
WHERE id = $1;

-- name: DeleteBindingByUserAndProvider :exec
DELETE FROM social_bindings
WHERE user_id = $1 AND provider = $2;

-- name: UpdateBindingTokens :exec
UPDATE social_bindings
SET access_token = $2,
    refresh_token = $3,
    token_expiry = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateBindingProfile :exec
UPDATE social_bindings
SET provider_email = $2,
    provider_name = $3,
    provider_avatar = $4,
    raw_profile = $5,
    verified_at = $6,
    updated_at = NOW()
WHERE id = $1;
