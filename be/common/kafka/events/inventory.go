package events

import "github.com/google/uuid"

const (
	EventInventoryReserved          = "inventory.reserved"
	EventInventoryReservationFailed = "inventory.reservation_fail"
)

// InventoryReservedPayload is emitted when inventory reservation succeeds for an order.
type InventoryReservedPayload struct {
	OrderID uuid.UUID `json:"order_id"`
}

// InventoryReservationFailedPayload is emitted when inventory reservation fails for an order.
type InventoryReservationFailedPayload struct {
	OrderID uuid.UUID `json:"order_id"`
	Message string    `json:"message"`
}
