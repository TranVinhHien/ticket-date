-- name: CreateOrder :one
INSERT INTO orders (
    id,
    user_id,
    order_code,
    total_amount,
    currency,
    status,
    expires_at,
    paid_at,
    created_by,
    created_at,
    updated_by,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $9, NOW()
)
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    id,
    order_id,
    ticket_type_id,
    ticket_type_name,
    sku,
    quantity,
    unit_price,
    subtotal,
    created_by,
    created_at,
    updated_by,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $9, NOW()
)
RETURNING *;

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
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetOrderByCode :one
SELECT * FROM orders
WHERE order_code = $1 AND deleted_at IS NULL;

-- name: GetOrderItemsByOrderID :many
SELECT * FROM order_items
WHERE order_id = $1 AND deleted_at IS NULL;

-- name: ListOrdersByUserID :many
SELECT * FROM orders
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOrdersByUserID :one
SELECT COUNT(*) FROM orders
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: AcquireOrderIdempotencyKey :one
WITH ins AS (
    INSERT INTO order_idempotency_keys (
        user_id,
        key,
        order_id,
        expires_at
    ) VALUES (
        $1, $2, $3, $4
    )
    ON CONFLICT (user_id, key) DO UPDATE
        SET order_id = EXCLUDED.order_id,
            created_at = NOW(),
            expires_at = EXCLUDED.expires_at
    WHERE order_idempotency_keys.expires_at <= NOW()
    RETURNING user_id, key, order_id, created_at, expires_at, TRUE AS acquired
)
SELECT user_id, key, order_id, created_at, expires_at, acquired FROM ins
UNION ALL
SELECT user_id, key, order_id, created_at, expires_at, FALSE AS acquired
FROM order_idempotency_keys
WHERE user_id = $1 AND key = $2 AND NOT EXISTS (SELECT 1 FROM ins);

-- name: DeleteOrderIdempotencyKey :exec
DELETE FROM order_idempotency_keys
WHERE user_id = $1 AND key = $2 AND order_id = $3;

-- name: ListOrdersByStatus :many
SELECT * FROM orders
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

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

-- name: MarkOrderReserved :one
UPDATE orders
SET
    status = 'RESERVED',
    updated_at = NOW()
WHERE id = $1
  AND status = 'PENDING'
  AND deleted_at IS NULL
RETURNING *;

-- name: MarkOrderFailed :one
UPDATE orders
SET
    status = 'FAILED',
    message = $2,
    updated_at = NOW()
WHERE id = $1
  AND status = 'PENDING'
  AND deleted_at IS NULL
RETURNING *;
