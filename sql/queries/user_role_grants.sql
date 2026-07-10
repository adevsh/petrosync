
-- =============================================================================
-- SECTION 13 — USER ROLE GRANTS
-- =============================================================================

-- name: GetActiveRolesForUser :many
SELECT *
FROM user_role_grants
WHERE user_id    = $1
  AND revoked_at IS NULL
ORDER BY role, scope_type;

-- name: GetActiveRoleForUserAndScope :one
SELECT *
FROM user_role_grants
WHERE user_id    = $1
  AND role       = $2
  AND scope_type = $3
  AND scope_id   = $4
  AND revoked_at IS NULL;

-- name: CheckUserHasRoleInScope :one
-- Returns TRUE if the user has the given role active for the given scope.
SELECT EXISTS (
    SELECT 1
    FROM user_role_grants
    WHERE user_id    = $1
      AND role       = $2
      AND scope_type = $3
      AND scope_id   = $4
      AND revoked_at IS NULL
) AS has_role;

-- name: CheckUserHasCompanyRole :one
SELECT EXISTS (
    SELECT 1
    FROM user_role_grants
    WHERE user_id    = $1
      AND role       = $2
      AND scope_type = 'COMPANY'
      AND revoked_at IS NULL
) AS has_role;

-- name: ListUsersWithRoleInScope :many
SELECT
    u.id,
    u.username,
    u.full_name,
    u.telegram_user_id,
    u.active,
    urg.granted_at
FROM user_role_grants urg
JOIN users u ON u.id = urg.user_id
WHERE urg.role       = $1
  AND urg.scope_type = $2
  AND urg.scope_id   = $3
  AND urg.revoked_at IS NULL
  AND u.active       = TRUE
ORDER BY u.full_name;

-- name: ListUsersWithCompanyRole :many
SELECT
    u.id,
    u.username,
    u.full_name,
    u.telegram_user_id,
    u.active,
    urg.granted_at
FROM user_role_grants urg
JOIN users u ON u.id = urg.user_id
WHERE urg.role       = $1
  AND urg.scope_type = 'COMPANY'
  AND urg.scope_id IS NULL
  AND urg.revoked_at IS NULL
  AND u.active       = TRUE
ORDER BY u.full_name;

-- name: GrantRole :one
INSERT INTO user_role_grants (user_id, role, scope_type, scope_id, granted_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, role, scope_type, scope_id)
DO UPDATE SET
    revoked_at = NULL,
    granted_by = EXCLUDED.granted_by,
    granted_at = NOW()
RETURNING *;

-- name: RevokeRole :exec
UPDATE user_role_grants
SET revoked_at = NOW()
WHERE user_id    = $1
  AND role       = $2
  AND scope_type = $3
  AND scope_id   = $4
  AND revoked_at IS NULL;

-- name: RevokeAllRolesForUser :exec
UPDATE user_role_grants
SET revoked_at = NOW()
WHERE user_id    = $1
  AND revoked_at IS NULL;
