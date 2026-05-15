-- name: GetProviderConfig :one
SELECT * FROM provider_configs
WHERE provider = $1
LIMIT 1;

-- name: ListProviderConfigs :many
SELECT * FROM provider_configs
ORDER BY provider ASC;

-- name: ListEnabledProviderConfigs :many
SELECT * FROM provider_configs
WHERE enabled = TRUE
ORDER BY provider ASC;

-- name: UpsertProviderConfig :one
INSERT INTO provider_configs (
    provider,
    display_name,
    enabled,
    client_id,
    client_secret,
    scopes,
    redirect_path,
    extra_config
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (provider) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    enabled = EXCLUDED.enabled,
    client_id = EXCLUDED.client_id,
    client_secret = EXCLUDED.client_secret,
    scopes = EXCLUDED.scopes,
    redirect_path = EXCLUDED.redirect_path,
    extra_config = EXCLUDED.extra_config,
    updated_at = NOW()
RETURNING *;

-- name: DeleteProviderConfig :exec
DELETE FROM provider_configs
WHERE provider = $1;
