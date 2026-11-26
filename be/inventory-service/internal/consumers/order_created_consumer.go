package consumers

import (
	"context"
	"fmt"
	"log"

	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
)

// RegisterOrderEventHandlers registers all order-related Kafka event handlers on the shared consumer.
func RegisterOrderEventHandlers(consumer commonKafka.Consumer, invService services.InventoryService) {
	if consumer == nil || invService == nil {
		return
	}
	consumer.RegisterHandler(events.TopicOrderEvents, NewOrderCreatedHandler(invService))
}

// NewOrderCreatedHandler returns a Kafka HandlerFunc that processes OrderCreated events.
func NewOrderCreatedHandler(invService services.InventoryService) commonKafka.HandlerFunc {
	return func(ctx context.Context, env *commonKafka.EventEnvelope) error {
		payload, err := commonKafka.DecodeData[events.OrderCreatedPayload](env)
		if err != nil {
			return fmt.Errorf("decode OrderCreated payload: %w", err)
		}

		log.Printf("[OrderCreatedConsumer] Processing OrderCreated event_id=%s order_id=%s items=%d",
			env.EventID, payload.ID, len(payload.Items))

		if err := invService.HandleOrderCreated(ctx, env.EventID, payload); err != nil {
			return fmt.Errorf("handle OrderCreated order_id=%s: %w", payload.ID, err)
		}

		log.Printf("[OrderCreatedConsumer] Successfully processed OrderCreated order_id=%s", payload.ID)
		return nil
	}
}
