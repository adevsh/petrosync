
-- =============================================================================
-- SECTION 25 — COMPARTMENT SEALS
-- =============================================================================

-- name: GetSealByTripAndCompartment :one
SELECT *
FROM compartment_seals
WHERE trip_id      = $1
  AND compartment_id = $2;

-- name: ListSealsByTrip :many
SELECT
    cs.*,
    vc.compartment_number,
    ui.full_name AS issued_by_name,
    uv.full_name AS verified_by_name
FROM compartment_seals    cs
JOIN vehicle_compartments vc ON vc.id = cs.compartment_id
JOIN users                ui ON ui.id = cs.issued_by
LEFT JOIN users           uv ON uv.id = cs.verified_by
WHERE cs.trip_id = $1
ORDER BY vc.compartment_number;

-- name: ListMismatchedSealsByTrip :many
SELECT *
FROM compartment_seals
WHERE trip_id             = $1
  AND verification_status IN ('MISMATCHED', 'BROKEN', 'MISSING');

-- name: IssueSeal :one
INSERT INTO compartment_seals (
    trip_id, compartment_id, seal_number_issued, issued_by
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: VerifySeal :one
UPDATE compartment_seals
SET
    seal_number_verified = $2,
    verified_by          = $3,
    verified_at          = NOW(),
    verification_status  = CASE
        WHEN seal_number_issued = $2 THEN 'INTACT'::seal_status_t
        ELSE 'MISMATCHED'::seal_status_t
    END,
    notes = $4
WHERE id = $1
RETURNING *;

-- name: RecordSealBreak :one
UPDATE compartment_seals
SET
    verified_by         = $2,
    verified_at         = NOW(),
    verification_status = $3,
    notes               = $4
WHERE id = $1
RETURNING *;

-- name: CountSealMismatchesByTrip :one
SELECT COUNT(*)::INT AS mismatch_count
FROM compartment_seals
WHERE trip_id = $1
  AND verification_status IN ('MISMATCHED', 'BROKEN', 'MISSING');
