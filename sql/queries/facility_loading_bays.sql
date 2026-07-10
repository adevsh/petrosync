
-- =============================================================================
-- SECTION 7 — FACILITY LOADING BAYS
-- =============================================================================

-- name: GetLoadingBay :one
SELECT *
FROM facility_loading_bays
WHERE id = $1;

-- name: GetLoadingBayByQRPayload :one
SELECT flb.*, rf.id AS refinery_id
FROM facility_loading_bays flb
JOIN refinery_facilities rf ON rf.id = flb.facility_id
WHERE flb.qr_payload = $1
  AND flb.active = TRUE;

-- name: ListLoadingBaysByFacility :many
SELECT *
FROM facility_loading_bays
WHERE facility_id = $1
  AND active = TRUE
ORDER BY bay_code;

-- name: CreateLoadingBay :one
INSERT INTO facility_loading_bays (
    facility_id, bay_code, qr_payload, fuel_type_code
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateLoadingBayFuelType :one
UPDATE facility_loading_bays
SET fuel_type_code = $2
WHERE id = $1
RETURNING *;

-- name: DeactivateLoadingBay :exec
UPDATE facility_loading_bays
SET active = FALSE
WHERE id = $1;

-- name: ValidateLoadingBayQR :one
-- Validates QR payload belongs to the expected facility and is active.
SELECT flb.id, flb.facility_id, flb.bay_code, flb.fuel_type_code
FROM facility_loading_bays flb
WHERE flb.qr_payload = $1
  AND flb.facility_id = $2
  AND flb.active = TRUE;
