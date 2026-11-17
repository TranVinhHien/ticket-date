-- name: CreateInventoryTransaction :one
INSERT INTO inventory_transactions (
    id,
    transaction_type,
    reference_type,
    reference_id,
    note,
    created_by,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, NOW()
)
RETURNING *;

-- name: CreateTicketInventory :one
INSERT INTO ticket_inventories (
    id,
    ticket_type_id,
    total_quantity,
    held_quantity,
    sold_quantity,
    created_by,
    created_at,
    updated_by,
    updated_at
) VALUES (
    $1, $2, $3, 0, 0, $4, NOW(), $4, NOW()
)
RETURNING *;

-- name: CreateInventoryTransactionItem :one
INSERT INTO inventory_transaction_items (
    id,
    inventory_transaction_id,
    ticket_inventory_id,
    quantity,
    created_at
) VALUES (
    $1, $2, $3, $4, NOW()
)
RETURNING *;

-- name: GetExistingTicketTypeIDs :many
SELECT ticket_type_id
FROM ticket_inventories
WHERE ticket_type_id = ANY($1::uuid[])
  AND deleted_at IS NULL;

-- name: GetInventoriesByTicketTypeIDs :many
SELECT *
FROM ticket_inventories
WHERE ticket_type_id = ANY($1::uuid[])
  AND deleted_at IS NULL;

-- name: GetInventoriesByTicketTypeIDsForUpdate :many
SELECT *
FROM ticket_inventories
WHERE ticket_type_id = ANY($1::uuid[])
  AND deleted_at IS NULL
FOR UPDATE;

-- name: GetInventoryByTicketTypeID :one
SELECT *
FROM ticket_inventories
WHERE ticket_type_id = $1
  AND deleted_at IS NULL;

-- name: UpdateTicketInventoryStock :one
UPDATE ticket_inventories
SET
    total_quantity = $2,
    updated_by = $3,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;
