-- name: GetSetting :one
SELECT * FROM global_settings
WHERE key = $1
LIMIT 1;

-- name: UpsertSetting :exec
INSERT INTO global_settings (key, value, description, updated_by, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW();

-- name: ListSettings :many
SELECT * FROM global_settings
ORDER BY key ASC;

-- name: DeleteSetting :exec
DELETE FROM global_settings
WHERE key = $1;
