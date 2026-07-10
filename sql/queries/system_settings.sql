
-- =============================================================================
-- SECTION 3 — SYSTEM SETTINGS
-- =============================================================================

-- name: GetGlobalSetting :one
SELECT *
FROM system_settings
WHERE key = $1
  AND facility_id IS NULL;

-- name: GetFacilitySetting :one
SELECT *
FROM system_settings
WHERE key = $1
  AND facility_id = $2;

-- Resolve effective value: facility-specific overrides global default.
-- name: GetEffectiveSetting :one
SELECT COALESCE(fac.value, gbl.value) AS value
FROM system_settings gbl
LEFT JOIN system_settings fac
       ON fac.key = gbl.key
      AND fac.facility_id = $2
WHERE gbl.key = $1
  AND gbl.facility_id IS NULL;

-- name: ListSettingsByFacility :many
SELECT *
FROM system_settings
WHERE facility_id = $1
ORDER BY key;

-- name: ListGlobalSettings :many
SELECT *
FROM system_settings
WHERE facility_id IS NULL
ORDER BY key;

-- name: UpsertGlobalSetting :one
INSERT INTO system_settings (facility_id, key, value, description, updated_by)
VALUES (NULL, $1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT idx_system_settings_global
DO UPDATE SET
    value      = EXCLUDED.value,
    description = COALESCE(EXCLUDED.description, system_settings.description),
    updated_by  = EXCLUDED.updated_by,
    updated_at  = NOW()
RETURNING *;

-- name: UpsertFacilitySetting :one
INSERT INTO system_settings (facility_id, key, value, description, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ON CONSTRAINT idx_system_settings_facility
DO UPDATE SET
    value       = EXCLUDED.value,
    description = COALESCE(EXCLUDED.description, system_settings.description),
    updated_by  = EXCLUDED.updated_by,
    updated_at  = NOW()
RETURNING *;

-- name: DeleteFacilitySetting :exec
DELETE FROM system_settings
WHERE key = $1
  AND facility_id = $2;
