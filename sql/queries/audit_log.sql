
-- =============================================================================
-- SECTION 31 — AUDIT LOG (APPEND-ONLY)
-- =============================================================================

-- name: InsertAuditLog :one
INSERT INTO audit_log (
    user_id, action, entity_type, entity_id,
    before_state, after_state, ip_address, user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, created_at;

-- name: ListAuditLogByEntity :many
SELECT
    al.*,
    u.username,
    u.full_name
FROM audit_log al
LEFT JOIN users u ON u.id = al.user_id
WHERE al.entity_type = $1
  AND al.entity_id   = $2
ORDER BY al.created_at DESC
LIMIT $3;

-- name: ListAuditLogByUser :many
SELECT *
FROM audit_log
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListAuditLogByAction :many
SELECT
    al.*,
    u.username,
    u.full_name
FROM audit_log al
LEFT JOIN users u ON u.id = al.user_id
WHERE al.action = $1
ORDER BY al.created_at DESC
LIMIT $2;
