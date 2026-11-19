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
