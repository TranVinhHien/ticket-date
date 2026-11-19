package handlers

import (
	"context"
	"log"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
	"github.com/google/uuid"
)

type InventoryGRPCHandler struct {
	pbInventory.UnimplementedInventoryServiceServer
	service services.InventoryService
}

func NewInventoryGRPCHandler(service services.InventoryService) *InventoryGRPCHandler {
	return &InventoryGRPCHandler{service: service}
}

// DeductStock processes atomic inventory deduction across multiple ticket types.
func (h *InventoryGRPCHandler) DeductStock(ctx context.Context, req *pbInventory.DeductStockRequest) (*pbInventory.DeductStockResponse, error) {
	log.Printf("[InventoryGRPCHandler.DeductStock] Incoming gRPC deduct request: %d items", len(req.GetItems()))
	if req == nil || len(req.Items) == 0 {
		log.Printf("[InventoryGRPCHandler.DeductStock] Validation failed: items is empty")
		return nil, grpcErrors.InvalidArgument("items cannot be empty")
	}

	items := make([]dto.DeductStockItemDTO, 0, len(req.Items))
	for _, item := range req.Items {
		if item == nil {
			log.Printf("[InventoryGRPCHandler.DeductStock] Validation failed: item is nil")
			return nil, grpcErrors.InvalidArgument("item cannot be nil")
		}
		if item.Quantity <= 0 {
			log.Printf("[InventoryGRPCHandler.DeductStock] Validation failed: invalid quantity %d", item.Quantity)
			return nil, grpcErrors.InvalidArgument("quantity must be greater than 0 for ticket_type_id: %s", item.TicketTypeId)
		}
		ticketTypeID, err := uuid.Parse(item.TicketTypeId)
		if err != nil || ticketTypeID == uuid.Nil {
			log.Printf("[InventoryGRPCHandler.DeductStock] Invalid ticketTypeID: %s", item.TicketTypeId)
			return nil, grpcErrors.InvalidArgument("invalid ticket_type_id format: %s", item.TicketTypeId)
		}
		items = append(items, dto.DeductStockItemDTO{
			TicketTypeID: ticketTypeID,
			Quantity:     item.Quantity,
		})
	}

	dtoReq := dto.DeductStockRequest{
		Items: items,
	}

	if err := h.service.DeductStock(ctx, dtoReq); err != nil {
		log.Printf("[InventoryGRPCHandler.DeductStock] Service error deducting stock: %v", err)
		return nil, err
	}

	log.Printf("[InventoryGRPCHandler.DeductStock] Successfully deducted stock for %d items", len(items))
	return &pbInventory.DeductStockResponse{
		Success: true,
	}, nil
}
