
-- =============================================================================
-- SECTION 17 — STATION QR CODES
-- =============================================================================

-- name: GetStationQRCode :one
SELECT *
FROM station_qr_codes
WHERE id = $1;

-- name: GetStationByQRPayload :one
-- Validates QR payload belongs to the expected station and is active.
SELECT
    sqr.id       AS qr_id,
    sqr.label,
    gs.id        AS station_id,
    gs.code      AS station_code,
    gs.name      AS station_name,
    gs.primary_facility_id
FROM station_qr_codes sqr
JOIN gas_stations      gs ON gs.id = sqr.station_id
WHERE sqr.qr_payload = $1
  AND sqr.active     = TRUE
  AND gs.active      = TRUE;

-- name: ListQRCodesByStation :many
SELECT *
FROM station_qr_codes
WHERE station_id = $1
ORDER BY label;

-- name: CreateStationQRCode :one
INSERT INTO station_qr_codes (station_id, qr_payload, label)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeactivateStationQRCode :exec
UPDATE station_qr_codes
SET active = FALSE
WHERE id = $1;

-- name: DeactivateAllStationQRCodes :exec
UPDATE station_qr_codes
SET active = FALSE
WHERE station_id = $1;
