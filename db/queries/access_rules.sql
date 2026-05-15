-- name: CreateAccessRule :one
INSERT INTO client_access_rules (
    client_id,
    rule_type,
    rule_value,
    effect,
    priority,
    description,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetAccessRuleByID :one
SELECT * FROM client_access_rules
WHERE id = $1
LIMIT 1;

-- name: ListAccessRulesByClient :many
SELECT * FROM client_access_rules
WHERE client_id = $1
ORDER BY priority DESC, created_at DESC;

-- name: ListActiveAccessRulesByClient :many
SELECT * FROM client_access_rules
WHERE client_id = $1 AND is_active = TRUE
ORDER BY priority DESC, created_at DESC;

-- name: DeleteAccessRule :exec
DELETE FROM client_access_rules
WHERE id = $1;

-- name: UpdateAccessRule :exec
UPDATE client_access_rules
SET rule_type = $2,
    rule_value = $3,
    effect = $4,
    priority = $5,
    description = $6,
    is_active = $7,
    updated_at = NOW()
WHERE id = $1;
