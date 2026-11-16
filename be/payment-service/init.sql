
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


-- PAYMENTS
CREATE INDEX idx_payments_order
    ON payments(order_id)
    WHERE deleted_at IS NULL;