package consumers

import (
	"context"
	"fmt"
	"log"

	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/services"
)

// RegisterInventoryEventHandlers registers all inventory-related Kafka event handlers on the shared consumer.
func RegisterInventoryEventHandlers(consumer commonKafka.Consumer, orderService services.OrderService) {
	if consumer == nil || orderService == nil {
		return
	}
	consumer.RegisterHandler(events.EventInventoryReserved, NewInventoryReservedHandler(orderService))
	consumer.RegisterHandler(events.EventInventoryReservationFailed, NewInventoryReservationFailedHandler(orderService))
}

// NewInventoryReservedHandler returns a Kafka HandlerFunc that processes inventory.reserved events.
func NewInventoryReservedHandler(orderService services.OrderService) commonKafka.HandlerFunc {
	return func(ctx context.Context, env *commonKafka.EventEnvelope) error {
		payload, err := commonKafka.DecodeData[events.InventoryReservedPayload](env)
		if err != nil {
			return fmt.Errorf("decode InventoryReserved payload: %w", err)
		}

		log.Printf("[InventoryConsumer] Processing %s event_id=%s order_id=%s",
			events.EventInventoryReserved, env.EventID, payload.OrderID)

		if err := orderService.HandleInventoryReserved(ctx, env.EventID, payload); err != nil {
			return fmt.Errorf("handle %s order_id=%s: %w", events.EventInventoryReserved, payload.OrderID, err)
		}

		return nil
	}
}

// NewInventoryReservationFailedHandler returns a Kafka HandlerFunc that processes inventory.reservation_fail events.
func NewInventoryReservationFailedHandler(orderService services.OrderService) commonKafka.HandlerFunc {
	return func(ctx context.Context, env *commonKafka.EventEnvelope) error {
		payload, err := commonKafka.DecodeData[events.InventoryReservationFailedPayload](env)
		if err != nil {
			return fmt.Errorf("decode InventoryReservationFailed payload: %w", err)
		}

		log.Printf("[InventoryConsumer] Processing %s event_id=%s order_id=%s message=%q",
			events.EventInventoryReservationFailed, env.EventID, payload.OrderID, payload.Message)

		if err := orderService.HandleInventoryReservationFailed(ctx, env.EventID, payload); err != nil {
			return fmt.Errorf("handle %s order_id=%s: %w", events.EventInventoryReservationFailed, payload.OrderID, err)
		}

		return nil
	}
}
