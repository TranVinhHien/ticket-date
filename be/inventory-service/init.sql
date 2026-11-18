

-- =========================================
-- 6. TICKET INVENTORIES
-- =========================================

CREATE TABLE ticket_inventories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    ticket_type_id UUID NOT NULL UNIQUE REFERENCES ticket_types(id),

    total_quantity INT NOT NULL,
    held_quantity INT NOT NULL DEFAULT 0,
    sold_quantity INT NOT NULL DEFAULT 0,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_ticket_inventory_quantity CHECK (
        total_quantity >= 0
        AND held_quantity >= 0
        AND sold_quantity >= 0
        AND held_quantity + sold_quantity <= total_quantity
    )
);


CREATE TABLE inventory_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    transaction_type VARCHAR(30) NOT NULL,

    reference_type VARCHAR(30),
    reference_id UUID,

    note TEXT,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_inventory_transaction_type CHECK (
        transaction_type IN (
            'INITIAL_STOCK',
            'HOLD',
            'RELEASE',
            'CONFIRM_SALE',
            'REFUND',
            'ADJUSTMENT'
        )
    )
);

CREATE TABLE inventory_transaction_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    inventory_transaction_id UUID NOT NULL
        REFERENCES inventory_transactions(id),

    ticket_inventory_id UUID NOT NULL
        REFERENCES ticket_inventories(id),

    quantity INT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_inventory_transaction_item_quantity
        CHECK (quantity > 0)
);

CREATE INDEX idx_inventory_transaction_items_transaction
    ON inventory_transaction_items(inventory_transaction_id);

CREATE INDEX idx_inventory_transaction_items_inventory
    ON inventory_transaction_items(ticket_inventory_id);

-- 8. idempotency_key table — consumer gate chống Kafka retry duplicate
CREATE TABLE  idempotency_key (
    token      VARCHAR(64) PRIMARY KEY NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
)
CREATE INDEX idx_idempotency_expires_at ON idempotency_key (expires_at);
