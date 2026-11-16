

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
