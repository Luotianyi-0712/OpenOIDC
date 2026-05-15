-- name: CreateSession :one
INSERT INTO user_sessions (
    user_id,
    token,
    ip_address,
    user_agent,
    device_info,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM user_sessions
WHERE token = $1 AND revoked_at IS NULL AND expires_at > NOW()
LIMIT 1;

-- name: GetSessionByID :one
SELECT * FROM user_sessions
WHERE id = $1
LIMIT 1;

-- name: ListSessionsByUser :many
SELECT * FROM user_sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
ORDER BY last_seen_at DESC;

-- name: TouchSession :exec
UPDATE user_sessions
SET last_seen_at = NOW()
WHERE token = $1;

-- name: DeleteSession :exec
UPDATE user_sessions
SET revoked_at = NOW()
WHERE id = $1;

-- name: DeleteSessionByToken :exec
UPDATE user_sessions
SET revoked_at = NOW()
WHERE token = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM user_sessions
WHERE expires_at < NOW() OR (revoked_at IS NOT NULL AND revoked_at < NOW() - INTERVAL '7 days');

-- name: DeleteSessionsByUser :exec
UPDATE user_sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;
