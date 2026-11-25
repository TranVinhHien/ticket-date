package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInventoryAlreadyExists = errors.New("inventory for one or more ticket types already exists")
	ErrInventoryNotFound      = errors.New("inventory not found for one or more ticket types")
	ErrInvalidQuantity        = errors.New("total quantity cannot be less than held + sold quantity")
	ErrEmptyBatch             = errors.New("batch items cannot be empty")
	ErrDuplicateTicketTypeID  = errors.New("duplicate ticket_type_id in request batch")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrInvalidTicketTypeID    = errors.New("invalid ticket_type_id")
)

type DeductStockItemDTO struct {
	TicketTypeID uuid.UUID `json:"ticket_type_id" binding:"required"`
	Quantity     int32     `json:"quantity" binding:"required,gt=0"`
}

type DeductStockRequest struct {
	Items []DeductStockItemDTO `json:"items" binding:"required,min=1,dive"`
}

type RestoreStockRequest struct {
	Items []DeductStockItemDTO `json:"items" binding:"required,min=1,dive"`
}

type CreateInventoryItem struct {
	TicketTypeID  uuid.UUID `json:"ticket_type_id" binding:"required"`
	TotalQuantity int32     `json:"total_quantity" binding:"min=0"`
}

type CreateInventoryBatchRequest struct {
	ReferenceType *string               `json:"reference_type"`
	ReferenceID   *uuid.UUID            `json:"reference_id"`
	Note          *string               `json:"note"`
	Items         []CreateInventoryItem `json:"items" binding:"required,min=1,dive"`
}

type UpdateInventoryItem struct {
	TicketTypeID  uuid.UUID `json:"ticket_type_id" binding:"required"`
	TotalQuantity int32     `json:"total_quantity" binding:"min=0"`
}

type UpdateInventoryBatchRequest struct {
	ReferenceType *string               `json:"reference_type"`
	ReferenceID   *uuid.UUID            `json:"reference_id"`
	Note          *string               `json:"note"`
	Items         []UpdateInventoryItem `json:"items" binding:"required,min=1,dive"`
}

type GetStockRequest struct {
	TicketTypeID uuid.UUID `json:"ticket_type_id" form:"ticket_type_id" binding:"required"`
}

type InventoryResponse struct {
	ID                uuid.UUID  `json:"id"`
	TicketTypeID      uuid.UUID  `json:"ticket_type_id"`
	TotalQuantity     int32      `json:"total_quantity"`
	HeldQuantity      int32      `json:"held_quantity"`
	SoldQuantity      int32      `json:"sold_quantity"`
	AvailableQuantity int32      `json:"available_quantity"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedBy         *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type StockResponse struct {
	TicketTypeID      uuid.UUID `json:"ticket_type_id"`
	TotalQuantity     int32     `json:"total_quantity,omitempty"`
	HeldQuantity      int32     `json:"held_quantity,omitempty"`
	SoldQuantity      int32     `json:"sold_quantity,omitempty"`
	AvailableQuantity int32     `json:"available_quantity"`
	HasStock          bool      `json:"has_stock"`
}
