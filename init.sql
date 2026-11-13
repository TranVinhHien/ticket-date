-- =========================================
-- 1. USERS & ROLES
-- =========================================

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) NOT NULL UNIQUE
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(30),
    avatar_url TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- =========================================
-- 2. ORGANIZERS
-- =========================================

CREATE TABLE organizers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id),
    organization_name VARCHAR(255) NOT NULL,
    description TEXT,
    tax_code VARCHAR(50),
    address TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
    country VARCHAR(100) DEFAULT 'Vietnam',
    capacity INT NOT NULL,
    description TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- =========================================
-- 4. events
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
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- =========================================
-- 5. PRODUCTS (SALES PHASES)
-- =========================================

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    concert_id UUID NOT NULL REFERENCES events(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sale_start_at TIMESTAMPTZ NOT NULL,
    sale_end_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- =========================================
-- 6. PRODUCT VARIANTS
-- =========================================

CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    price BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',
    total_quantity INT NOT NULL,
    max_per_order INT NOT NULL DEFAULT 4,
    attributes JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_variant_price CHECK (price >= 0),
    CONSTRAINT chk_variant_quantity CHECK (total_quantity >= 0),
    CONSTRAINT chk_variant_max_order CHECK (max_per_order > 0)
);


-- =========================================
-- 7. VARIANT INVENTORIES
-- =========================================

CREATE TABLE variant_inventories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL UNIQUE REFERENCES product_variants(id),
    total_quantity INT NOT NULL,
    held_quantity INT NOT NULL DEFAULT 0,
    sold_quantity INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_inventory_quantity CHECK (
        total_quantity >= 0
        AND held_quantity >= 0
        AND sold_quantity >= 0
        AND held_quantity + sold_quantity <= total_quantity
    )
);


-- =========================================
-- 8. ORDERS
-- =========================================

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    order_code VARCHAR(50) NOT NULL UNIQUE,
    total_amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_order_amount CHECK (total_amount >= 0)
);


-- =========================================
-- 9. ORDER ITEMS
-- =========================================

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    variant_name VARCHAR(150) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_order_item_quantity CHECK (quantity > 0),
    CONSTRAINT chk_order_item_price CHECK (unit_price >= 0),
    CONSTRAINT chk_order_item_subtotal CHECK (subtotal >= 0)
);


-- =========================================
-- 10. PAYMENTS
-- =========================================

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    payment_code VARCHAR(100) UNIQUE,
    provider VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'VND',
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_payment_amount CHECK (amount >= 0)
);


-- =========================================
-- 11. ISSUED TICKETS
-- =========================================

CREATE TABLE issued_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id UUID NOT NULL REFERENCES order_items(id),
    ticket_code VARCHAR(100) NOT NULL UNIQUE,
    qr_token UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    status VARCHAR(30) NOT NULL DEFAULT 'VALID',
    checked_in_at TIMESTAMPTZ,
    checked_in_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- =========================================
-- 12. INDEXES
-- =========================================

CREATE INDEX idx_venues_organizer
    ON venues(organizer_id);

CREATE INDEX idx_events_organizer
    ON events(organizer_id);

CREATE INDEX idx_events_venue
    ON events(venue_id);

CREATE INDEX idx_events_status_start
    ON events(status, start_at);

CREATE INDEX idx_products_concert
    ON products(concert_id);

CREATE INDEX idx_products_sale_period
    ON products(sale_start_at, sale_end_at);

CREATE INDEX idx_variants_product
    ON product_variants(product_id);

CREATE INDEX idx_orders_user_created
    ON orders(user_id, created_at DESC);

CREATE INDEX idx_orders_status_expires
    ON orders(status, expires_at);

CREATE INDEX idx_order_items_order
    ON order_items(order_id);

CREATE INDEX idx_order_items_variant
    ON order_items(variant_id);

CREATE INDEX idx_payments_order
    ON payments(order_id);

CREATE INDEX idx_issued_tickets_order_item
    ON issued_tickets(order_item_id);

CREATE INDEX idx_issued_tickets_status
    ON issued_tickets(status);