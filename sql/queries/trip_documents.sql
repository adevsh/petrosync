
-- =============================================================================
-- SECTION 28 — TRIP DOCUMENTS
-- =============================================================================

-- name: GetTripDocument :one
SELECT *
FROM trip_documents
WHERE id = $1;

-- name: GetTripDocumentByType :one
SELECT *
FROM trip_documents
WHERE trip_id      = $1
  AND document_type = $2;

-- name: ListDocumentsByTrip :many
SELECT *
FROM trip_documents
WHERE trip_id = $1
ORDER BY generated_at;

-- name: CreateTripDocument :one
INSERT INTO trip_documents (
    trip_id, document_type, document_number,
    garage_object_key, generated_by
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateTripDocumentKey :one
-- Called if a document is regenerated (e.g., after variance resolution).
UPDATE trip_documents
SET
    garage_object_key = $2,
    generated_at      = NOW()
WHERE id = $1
RETURNING *;
