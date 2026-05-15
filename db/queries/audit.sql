-- name: CreateAuditLog :exec
INSERT INTO audit_log (
    user_id,
    actor_id,
    action,
    resource_type,
    resource_id,
    ip_address,
    user_agent,
    status,
    error_message,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: ListAuditLogs :many
SELECT * FROM audit_log
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id')::uuid)
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type')::text)
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*)::bigint AS count
FROM audit_log
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id')::uuid)
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type')::text);

-- name: CreateSecurityLevelChange :exec
INSERT INTO security_level_changes (
    user_id,
    old_level,
    new_level,
    reason,
    rule_id,
    changed_by,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: ListSecurityLevelChanges :many
SELECT * FROM security_level_changes
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id')::uuid)
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: CountSecurityLevelChanges :one
SELECT COUNT(*)::bigint AS count
FROM security_level_changes
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id')::uuid);
