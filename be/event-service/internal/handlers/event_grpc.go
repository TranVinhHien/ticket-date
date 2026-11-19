package handlers

import (
	"context"
	"log"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	pbEvent "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/event"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/services"
	"github.com/google/uuid"
)

type EventGRPCHandler struct {
	pbEvent.UnimplementedEventServiceServer
	eventService *services.EventService
}

func NewEventGRPCHandler(eventService *services.EventService) *EventGRPCHandler {
	return &EventGRPCHandler{
		eventService: eventService,
	}
}

// GetTicketTypes retrieves details of ticket types by a list of UUIDs.
func (h *EventGRPCHandler) GetTicketTypes(ctx context.Context, req *pbEvent.GetTicketTypesRequest) (*pbEvent.GetTicketTypesResponse, error) {
	log.Printf("[EventGRPCHandler.GetTicketTypes] Incoming request for ticketTypeIDs=%v", req.GetTicketTypeIds())
	if req == nil || len(req.TicketTypeIds) == 0 {
		log.Printf("[EventGRPCHandler.GetTicketTypes] Validation failed: ticket_type_ids is empty")
		return nil, grpcErrors.InvalidArgument("ticket_type_ids cannot be empty")
	}

	uuids := make([]uuid.UUID, 0, len(req.TicketTypeIds))
	for _, idStr := range req.TicketTypeIds {
		id, err := uuid.Parse(idStr)
		if err != nil || id == uuid.Nil {
			log.Printf("[EventGRPCHandler.GetTicketTypes] Invalid UUID format: %s", idStr)
			return nil, grpcErrors.InvalidArgument("invalid ticket_type_id format: %s", idStr)
		}
		uuids = append(uuids, id)
	}

	ticketTypes, err := h.eventService.GetTicketTypesByIDs(ctx, uuids)
	if err != nil {
		log.Printf("[EventGRPCHandler.GetTicketTypes] Service error: %v", err)
		return nil, grpcErrors.Internal("failed to fetch ticket types: %v", err)
	}

	details := make([]*pbEvent.TicketTypeDetail, 0, len(ticketTypes))
	for _, tt := range ticketTypes {
		desc := ""
		if tt.Description != nil {
			desc = *tt.Description
		}
		details = append(details, &pbEvent.TicketTypeDetail{
			Id:          tt.ID.String(),
			EventId:     tt.EventID.String(),
			Sku:         tt.SKU,
			Name:        tt.Name,
			Description: desc,
			Price:       tt.Price,
			Currency:    tt.Currency,
			MaxPerOrder: int32(tt.MaxPerOrder),
			Status:      string(tt.Status),
		})
	}

	log.Printf("[EventGRPCHandler.GetTicketTypes] Returning %d ticket type details", len(details))
	return &pbEvent.GetTicketTypesResponse{
		TicketTypes: details,
	}, nil
}
