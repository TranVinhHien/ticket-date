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

-- name: TryAcquireIdempotencyKey :one
INSERT INTO idempotency_key (token, expires_at)
VALUES ($1, $2)
ON CONFLICT (token) DO NOTHING
RETURNING token;

-- name: HoldTicketInventoryStock :one
UPDATE ticket_inventories
SET
    held_quantity = held_quantity + $2,
    updated_at = NOW()
WHERE ticket_type_id = $1
  AND deleted_at IS NULL
  AND held_quantity + sold_quantity + $2 <= total_quantity
RETURNING *;

-- name: HoldTicketInventoryStockBatch :many
WITH requested AS (
    SELECT
        unnest($1::uuid[]) AS ticket_type_id,
        unnest($2::int[]) AS qty
    ORDER BY 1
)
UPDATE ticket_inventories ti
SET
    held_quantity = ti.held_quantity + r.qty,
    updated_at = NOW()
FROM requested r
WHERE ti.ticket_type_id = r.ticket_type_id
  AND ti.deleted_at IS NULL
  AND ti.held_quantity + ti.sold_quantity + r.qty <= ti.total_quantity
RETURNING ti.*;

-- name: CreateInventoryTransactionItemBatch :exec
INSERT INTO inventory_transaction_items (
    id,
    inventory_transaction_id,
    ticket_inventory_id,
    quantity,
    created_at
)
SELECT
    unnest($1::uuid[]),
    $2,
    unnest($3::uuid[]),
    unnest($4::int[]),
    NOW();

-- name: CreateOutboxEvent :one
INSERT INTO outbox_event (
    aggregate_id,
    event_type,
    payload,
    status,
    created_at
) VALUES (
    $1, $2, $3, $4, NOW()
)
ON CONFLICT (aggregate_id, event_type) DO NOTHING
RETURNING *;

-- name: ClaimPendingOutboxEvents :many
WITH cte AS (
    SELECT id FROM outbox_event
    WHERE status = 0
    ORDER BY created_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_event o
SET status = 1, updated_at = NOW()
FROM cte
WHERE o.id = cte.id
RETURNING o.*;

-- name: MarkOutboxEventSuccess :one
UPDATE outbox_event
SET
    status = 2,
    published_at = NOW(),
    last_error = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkOutboxEventFailed :one
UPDATE outbox_event
SET
    retry_count = retry_count + $2,
    status = 3,
    last_error = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkOutboxEventRetryOrFailed :one
UPDATE outbox_event
SET
    retry_count = retry_count + 1,
    status = CASE WHEN retry_count + 1 >= $2 THEN 3 ELSE 0 END,
    last_error = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RecoverStaleSendingOutboxEvents :execrows
UPDATE outbox_event
SET
    retry_count = retry_count + 1,
    status = CASE WHEN retry_count + 1 >= $1 THEN 3 ELSE 0 END,
    last_error = 'recovered from stale SENDING state',
    updated_at = NOW()
WHERE status = 1
  AND updated_at <= NOW() - ($2::bigint * INTERVAL '1 second');
