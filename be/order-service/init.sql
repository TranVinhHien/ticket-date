

-- =========================================
-- 7. ORDERS
-- =========================================

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL,
    order_code VARCHAR(50) NOT NULL UNIQUE,

    total_amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',

    status VARCHAR(30) NOT NULL DEFAULT 'PENDING', -- PENDING, PAID, CANCELLED, EXPIRED, REFUNDED

    expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_order_amount CHECK (total_amount >= 0)
);


-- =========================================
-- 8. ORDER ITEMS
-- =========================================

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    order_id UUID NOT NULL REFERENCES orders(id),
    ticket_type_id UUID NOT NULL REFERENCES ticket_types(id),

    ticket_type_name VARCHAR(150) NOT NULL,
    sku VARCHAR(100) NOT NULL,

    quantity INT NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_order_item_quantity CHECK (quantity > 0),
    CONSTRAINT chk_order_item_price CHECK (unit_price >= 0),
    CONSTRAINT chk_order_item_subtotal CHECK (subtotal >= 0),
    CONSTRAINT chk_order_item_subtotal_calculation
        CHECK (subtotal = quantity * unit_price)
);

-- ORDERS
CREATE INDEX idx_orders_user_created
    ON orders(user_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_orders_status_expires
    ON orders(status, expires_at)
    WHERE deleted_at IS NULL;


-- ORDER ITEMS
CREATE INDEX idx_order_items_order
    ON order_items(order_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_order_items_ticket_type
    ON order_items(ticket_type_id)
    WHERE deleted_at IS NULL;