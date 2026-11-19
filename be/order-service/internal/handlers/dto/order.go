package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyOrderItems   = errors.New("order must contain at least one item")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
	ErrInvalidUnitPrice  = errors.New("unit price cannot be negative")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidUserID     = errors.New("valid user id is required")
)

type CreateOrderItemRequest struct {
	TicketTypeID   uuid.UUID `json:"ticket_type_id" binding:"required"`
	TicketTypeName string    `json:"ticket_type_name" binding:"required"`
	Sku            string    `json:"sku" binding:"required"`
	Quantity       int32     `json:"quantity" binding:"required,gt=0"`
	UnitPrice      int64     `json:"unit_price" binding:"min=0"`
}

type CreateOrderRequest struct {
	Currency string                   `json:"currency"`
	Items    []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type OrderItemResponse struct {
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

type OrderResponse struct {
	ID          uuid.UUID           `json:"id"`
	UserID      uuid.UUID           `json:"user_id"`
	OrderCode   string              `json:"order_code"`
	TotalAmount int64               `json:"total_amount"`
	Currency    string              `json:"currency"`
	Status      string              `json:"status"`
	ExpiresAt   *time.Time          `json:"expires_at"`
	PaidAt      *time.Time          `json:"paid_at"`
	Items       []OrderItemResponse `json:"items,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
