-- name: CreateUser :one
INSERT INTO users (
    email,
    email_verified,
    password_hash,
    display_name,
    alias,
    avatar_url,
    phone,
    phone_verified,
    security_level,
    role,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByAlias :one
SELECT * FROM users
WHERE alias = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE phone = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateUser :exec
UPDATE users
SET
    email = $2,
    email_verified = $3,
    display_name = $4,
    alias = $5,
    avatar_url = $6,
    phone = $7,
    phone_verified = $8,
    status = $9,
    role = $10,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserSecurityLevel :exec
UPDATE users
SET security_level = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT *
FROM users
WHERE deleted_at IS NULL
  AND (
    sqlc.narg('search')::text IS NULL
    OR email ILIKE '%' || sqlc.narg('search')::text || '%'
    OR display_name ILIKE '%' || sqlc.narg('search')::text || '%'
    OR alias ILIKE '%' || sqlc.narg('search')::text || '%'
  )
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*)::bigint AS count
FROM users
WHERE deleted_at IS NULL
  AND (
    sqlc.narg('search')::text IS NULL
    OR email ILIKE '%' || sqlc.narg('search')::text || '%'
    OR display_name ILIKE '%' || sqlc.narg('search')::text || '%'
    OR alias ILIKE '%' || sqlc.narg('search')::text || '%'
  )
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text);

-- name: DeleteUser :exec
UPDATE users
SET deleted_at = NOW(),
    status = 'deleted',
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
