-- =========================================
-- 2. ORGANIZERS
-- =========================================

CREATE TABLE organizers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organization_name VARCHAR(255) NOT NULL,
    description TEXT,
    logo_url TEXT,
    tax_code VARCHAR(50),
    address TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',-- PENDING, ACTIVE, REJECTED, SUSPENDED

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ
);


-- =========================================
-- 3. VENUES
-- =========================================

CREATE TABLE venues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organizer_id UUID NOT NULL REFERENCES organizers(id),

    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    city VARCHAR(100),
    country VARCHAR(100) NOT NULL DEFAULT 'Vietnam',

    map_url TEXT,
    latitude DECIMAL(10, 7),
    longitude DECIMAL(10, 7),

    capacity INT NOT NULL,
    description TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',-- ACTIVE, INACTIVE, SUSPENDED

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_venue_capacity CHECK (capacity > 0),
    CONSTRAINT chk_venue_latitude CHECK (
        latitude IS NULL OR latitude BETWEEN -90 AND 90
    ),
    CONSTRAINT chk_venue_longitude CHECK (
        longitude IS NULL OR longitude BETWEEN -180 AND 180
    )
);


-- =========================================
-- 4. EVENTS
-- =========================================

CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organizer_id UUID NOT NULL REFERENCES organizers(id),
    venue_id UUID NOT NULL REFERENCES venues(id),

    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    poster_url TEXT,

    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',-- DRAFT, PUBLISHED, CANCELLED, COMPLETED

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_event_period CHECK (end_at > start_at)
);


-- =========================================
-- 5. TICKET TYPES
-- =========================================

CREATE TABLE ticket_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    event_id UUID NOT NULL REFERENCES events(id),

    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,

    price BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',

    max_per_order INT NOT NULL DEFAULT 4,

    sale_start_at TIMESTAMPTZ NOT NULL,
    sale_end_at TIMESTAMPTZ NOT NULL,

    attributes JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- DRAFT, ACTIVE, PAUSED, ENDED

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_ticket_price CHECK (price >= 0),
    CONSTRAINT chk_ticket_max_order CHECK (max_per_order > 0),
    CONSTRAINT chk_ticket_sale_period CHECK (sale_end_at > sale_start_at)
);


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


-- =========================================
-- 9. PAYMENTS
-- =========================================

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    order_id UUID NOT NULL REFERENCES orders(id),

    payment_code VARCHAR(100) UNIQUE,
    provider VARCHAR(50) NOT NULL,

    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',

    status VARCHAR(30) NOT NULL DEFAULT 'PENDING', -- PENDING, SUCCESS, FAILED, REFUNDED, PROCESSING, CANCELLED
    paid_at TIMESTAMPTZ,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_payment_amount CHECK (amount >= 0)
);


-- =========================================
-- 10. ISSUED TICKETS
-- =========================================

CREATE TABLE issued_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    order_item_id UUID NOT NULL REFERENCES order_items(id),

    ticket_code VARCHAR(100) NOT NULL UNIQUE,
    qr_token UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

    status VARCHAR(30) NOT NULL DEFAULT 'VALID', -- VALID, USED, EXPIRED, CANCELLED, REFUNDED

    checked_in_at TIMESTAMPTZ,
    checked_in_by UUID,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ
);


-- =========================================
-- 11. INDEXES
-- =========================================

-- VENUES
CREATE INDEX idx_venues_organizer
    ON venues(organizer_id)
    WHERE deleted_at IS NULL;


-- EVENTS
CREATE INDEX idx_events_organizer
    ON events(organizer_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_events_venue
    ON events(venue_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_events_status_start
    ON events(status, start_at)
    WHERE deleted_at IS NULL;


-- TICKET TYPES
CREATE INDEX idx_ticket_types_event
    ON ticket_types(event_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_ticket_types_status_sale
    ON ticket_types(status, sale_start_at, sale_end_at)
    WHERE deleted_at IS NULL;


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


-- PAYMENTS
CREATE INDEX idx_payments_order
    ON payments(order_id)
    WHERE deleted_at IS NULL;


-- ISSUED TICKETS
CREATE INDEX idx_issued_tickets_order_item
    ON issued_tickets(order_item_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_issued_tickets_status
    ON issued_tickets(status)
    WHERE deleted_at IS NULL;