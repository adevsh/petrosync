
-- =============================================================================
-- SECTION 8 — FACILITY STORAGE TANKS
-- =============================================================================

-- name: GetStorageTank :one
SELECT *
FROM facility_storage_tanks
WHERE id = $1;

-- name: GetStorageTankByFacilityAndFuel :one
SELECT *
FROM facility_storage_tanks
WHERE facility_id = $1
  AND fuel_type_code = $2
  AND active = TRUE;

-- name: ListStorageTanksByFacility :many
SELECT *
FROM facility_storage_tanks
WHERE facility_id = $1
  AND active = TRUE
ORDER BY fuel_type_code;

-- name: GetStorageTankAvailableVolume :one
SELECT
    id,
    facility_id,
    fuel_type_code,
    capacity_l,
    current_volume_l,
    reserved_volume_l,
    (current_volume_l - reserved_volume_l) AS available_volume_l
FROM facility_storage_tanks
WHERE id = $1
  AND active = TRUE;

-- name: CreateStorageTank :one
INSERT INTO facility_storage_tanks (
    facility_id, tank_code, fuel_type_code,
    capacity_l, current_volume_l, min_operational_l
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ReserveStorageTankVolume :one
-- Atomically reserve volume at DO approval. Fails if insufficient available.
UPDATE facility_storage_tanks
SET
    reserved_volume_l = reserved_volume_l + $2,
    last_updated_at   = NOW()
WHERE id = $1
  AND active = TRUE
  AND (current_volume_l - reserved_volume_l) >= $2
RETURNING *;

-- name: ReleaseStorageTankReservation :one
-- Release reservation (on DO cancellation or manual override).
UPDATE facility_storage_tanks
SET
    reserved_volume_l = GREATEST(0, reserved_volume_l - $2),
    last_updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: DeductStorageTankVolume :one
-- Deduct from current volume at loading completion (simultaneously clears reservation).
UPDATE facility_storage_tanks
SET
    current_volume_l  = current_volume_l - $2,
    reserved_volume_l = GREATEST(0, reserved_volume_l - $2),
    last_updated_at   = NOW()
WHERE id = $1
  AND active = TRUE
  AND current_volume_l >= $2
RETURNING *;

-- name: CreditStorageTankVolume :one
-- Credit volume on return-to-facility trip delivery.
UPDATE facility_storage_tanks
SET
    current_volume_l = current_volume_l + $2,
    last_updated_at  = NOW()
WHERE id = $1
  AND active = TRUE
RETURNING *;

-- name: UpdateStorageTankVolume :one
-- Manual correction by SYSTEM_ADMIN (audited separately via audit_log).
UPDATE facility_storage_tanks
SET
    current_volume_l = $2,
    last_updated_at  = NOW()
WHERE id = $1
RETURNING *;
