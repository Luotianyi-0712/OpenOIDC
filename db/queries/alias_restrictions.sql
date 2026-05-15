-- name: CreateAliasRestriction :one
INSERT INTO alias_restrictions (
    pattern,
    restriction_type,
    description,
    created_by
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetAliasRestrictionByPattern :one
SELECT * FROM alias_restrictions
WHERE pattern = $1
LIMIT 1;

-- name: ListAliasRestrictions :many
SELECT * FROM alias_restrictions
ORDER BY created_at DESC;

-- name: ListAliasRestrictionsByType :many
SELECT * FROM alias_restrictions
WHERE restriction_type = $1
ORDER BY pattern ASC;

-- name: DeleteAliasRestriction :exec
DELETE FROM alias_restrictions
WHERE id = $1;
