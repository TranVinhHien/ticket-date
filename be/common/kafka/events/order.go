package events

import (
	"time"

	"github.com/google/uuid"
)

const (
	TopicOrderEvents = "order.events"
)

// OrderCreatedItem represents a line item inside an OrderCreated event payload.
type OrderCreatedItem struct {
	ID             uuid.UUID `json:"id"`
	OrderID        uuid.UUID `json:"order_id"`
	TicketTypeID   uuid.UUID `json:"ticket_type_id"`
	TicketTypeName string    `json:"ticket_type_name"`
	Sku            string    `json:"sku"`
	Quantity       int32     `json:"quantity"`
	UnitPrice      int64     `json:"unit_price"`
	Subtotal       int64     `json:"subtotal"`
	CreatedAt      time.Time `json:"created_at"`
}

// OrderCreatedPayload matches the JSON structure persisted by order-service in outbox_event.
type OrderCreatedPayload struct {
	ID          uuid.UUID          `json:"id"`
	UserID      uuid.UUID          `json:"user_id"`
	OrderCode   string             `json:"order_code"`
	TotalAmount int64              `json:"total_amount"`
	Currency    string             `json:"currency"`
	Status      string             `json:"status"`
	ExpiresAt   *time.Time         `json:"expires_at,omitempty"`
	PaidAt      *time.Time         `json:"paid_at,omitempty"`
	Items       []OrderCreatedItem `json:"items"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
