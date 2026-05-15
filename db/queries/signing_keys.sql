-- name: CreateSigningKey :one
INSERT INTO signing_keys (
    kid,
    algorithm,
    use_type,
    public_key,
    private_key,
    is_current,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetSigningKeyByKID :one
SELECT * FROM signing_keys
WHERE kid = $1
LIMIT 1;

-- name: GetCurrentSigningKey :one
SELECT * FROM signing_keys
WHERE is_current = TRUE
LIMIT 1;

-- name: ListSigningKeys :many
SELECT * FROM signing_keys
ORDER BY activated_at DESC;

-- name: ListActiveSigningKeys :many
SELECT * FROM signing_keys
WHERE rotated_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY activated_at DESC;

-- name: UnsetCurrentSigningKey :exec
UPDATE signing_keys
SET is_current = FALSE,
    rotated_at = COALESCE(rotated_at, NOW())
WHERE is_current = TRUE;

-- name: RotateSigningKey :exec
UPDATE signing_keys
SET is_current = FALSE,
    rotated_at = NOW()
WHERE id = $1;

-- name: SetCurrentSigningKey :exec
UPDATE signing_keys
SET is_current = TRUE
WHERE id = $1;

-- name: DeleteSigningKey :exec
DELETE FROM signing_keys
WHERE id = $1;

-- name: CreatePhoneVerification :one
INSERT INTO phone_verifications (
    user_id,
    phone,
    code,
    purpose,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetLatestPhoneVerification :one
SELECT * FROM phone_verifications
WHERE phone = $1 AND purpose = $2 AND verified = FALSE AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementPhoneVerificationAttempts :exec
UPDATE phone_verifications
SET attempts = attempts + 1
WHERE id = $1;

-- name: MarkPhoneVerificationVerified :exec
UPDATE phone_verifications
SET verified = TRUE,
    verified_at = NOW()
WHERE id = $1;

-- name: DeleteExpiredPhoneVerifications :exec
DELETE FROM phone_verifications
WHERE expires_at < NOW();
