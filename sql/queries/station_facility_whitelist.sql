
-- =============================================================================
-- SECTION 16 — STATION FACILITY WHITELIST
-- =============================================================================

-- name: ListFacilitiesForStation :many
SELECT
    sfw.facility_id,
    rf.code AS facility_code,
    rf.name AS facility_name,
    r.name  AS refinery_name,
    (gs.primary_facility_id = sfw.facility_id) AS is_primary
FROM station_facility_whitelist sfw
JOIN refinery_facilities rf ON rf.id = sfw.facility_id
JOIN refineries           r  ON r.id  = rf.refinery_id
JOIN gas_stations         gs ON gs.id = sfw.station_id
WHERE sfw.station_id = $1;

-- name: CheckFacilityCanServeStation :one
SELECT EXISTS (
    SELECT 1
    FROM station_facility_whitelist
    WHERE station_id  = $1
      AND facility_id = $2
) AS can_serve;

-- name: AddFacilityToStationWhitelist :exec
INSERT INTO station_facility_whitelist (station_id, facility_id)
VALUES ($1, $2)
ON CONFLICT (station_id, facility_id) DO NOTHING;

-- name: RemoveFacilityFromStationWhitelist :exec
DELETE FROM station_facility_whitelist
WHERE station_id  = $1
  AND facility_id = $2;
