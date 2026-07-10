
-- =============================================================================
-- SECTION 20 — DELIVERY ORDER ITEMS
-- =============================================================================

-- name: GetDeliveryOrderItem :one
SELECT *
FROM delivery_order_items
WHERE id = $1;

-- name: ListDOItemsByDO :many
SELECT
    doi.*,
    ft.name     AS fuel_type_name,
    ft.category AS fuel_category
FROM delivery_order_items doi
JOIN fuel_types ft ON ft.code = doi.fuel_type_code
WHERE doi.do_id = $1
ORDER BY doi.id;

-- name: CreateDeliveryOrderItem :one
INSERT INTO delivery_order_items (
    do_id, fuel_type_code, requested_volume_l
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: AssignCompartmentToDOItem :one
UPDATE delivery_order_items
SET
    compartment_id = $2,
    updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDOItemAllocatedVolume :one
UPDATE delivery_order_items
SET
    allocated_volume_l = $2,
    updated_at         = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDOItem :exec
DELETE FROM delivery_order_items
WHERE id = $1;
