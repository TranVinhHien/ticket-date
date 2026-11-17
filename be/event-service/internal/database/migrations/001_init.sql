-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =========================================
-- 1. ORGANIZERS
-- =========================================

CREATE TABLE IF NOT EXISTS organizers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organization_name VARCHAR(255) NOT NULL,
    description TEXT,
    logo_url TEXT,
    tax_code VARCHAR(50),
    address TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, INACTIVE

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ
);


-- =========================================
-- 2. VENUES
-- =========================================

CREATE TABLE IF NOT EXISTS venues (
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
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, INACTIVE, SUSPENDED

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
-- 3. EVENTS
-- =========================================

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organizer_id UUID NOT NULL REFERENCES organizers(id),
    venue_id UUID NOT NULL REFERENCES venues(id),

    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    poster_url TEXT,

    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- DRAFT, PUBLISHED, CANCELLED, COMPLETED

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_by UUID,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_event_period CHECK (end_at > start_at)
);


-- =========================================
-- 4. TICKET TYPES
-- =========================================

CREATE TABLE IF NOT EXISTS ticket_types (
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


-- INDEXES

CREATE INDEX IF NOT EXISTS idx_venues_organizer
    ON venues(organizer_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_events_organizer
    ON events(organizer_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_events_venue
    ON events(venue_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_events_status_start
    ON events(status, start_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ticket_types_event
    ON ticket_types(event_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ticket_types_status_sale
    ON ticket_types(status, sale_start_at, sale_end_at)
    WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ticket_types;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS venues;
DROP TABLE IF EXISTS organizers;
-- +goose StatementEnd
