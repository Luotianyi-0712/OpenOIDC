-- name: CreateRule :one
INSERT INTO security_level_rules (
    name,
    description,
    level,
    priority,
    conditions,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetRuleByID :one
SELECT * FROM security_level_rules
WHERE id = $1
LIMIT 1;

-- name: ListActiveRules :many
SELECT * FROM security_level_rules
WHERE is_active = TRUE
ORDER BY level DESC, priority DESC;

-- name: ListAllRules :many
SELECT * FROM security_level_rules
ORDER BY level DESC, priority DESC, created_at DESC;

-- name: UpdateRule :exec
UPDATE security_level_rules
SET name = $2,
    description = $3,
    level = $4,
    priority = $5,
    conditions = $6,
    is_active = $7,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteRule :exec
DELETE FROM security_level_rules
WHERE id = $1;
