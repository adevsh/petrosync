
-- =============================================================================
-- SECTION 1 — REGIONS
-- =============================================================================

-- name: GetRegion :one
SELECT code, name, created_at
FROM regions
WHERE code = $1;

-- name: ListRegions :many
SELECT code, name, created_at
FROM regions
ORDER BY name;
