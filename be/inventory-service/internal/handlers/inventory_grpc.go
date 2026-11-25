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
	var rawItems []*pbInventory.DeductItem
	if req != nil {
		rawItems = req.Items
	}
	log.Printf("[InventoryGRPCHandler.DeductStock] Incoming gRPC deduct request: %d items", len(rawItems))

	items, err := parseStockItems(rawItems, func(i *pbInventory.DeductItem) (string, int32, bool) {
		if i == nil {
			return "", 0, false
		}
		return i.TicketTypeId, i.Quantity, true
	})
	if err != nil {
		log.Printf("[InventoryGRPCHandler.DeductStock] Validation failed: %v", err)
		return nil, err
	}

	if err := h.service.DeductStock(ctx, dto.DeductStockRequest{Items: items}); err != nil {
		log.Printf("[InventoryGRPCHandler.DeductStock] Service error deducting stock: %v", err)
		return nil, err
	}

	log.Printf("[InventoryGRPCHandler.DeductStock] Successfully deducted stock for %d items", len(items))
	return &pbInventory.DeductStockResponse{
		Success: true,
	}, nil
}

// RestoreStock compensates previously deducted inventory across multiple ticket types.
func (h *InventoryGRPCHandler) RestoreStock(ctx context.Context, req *pbInventory.RestoreStockRequest) (*pbInventory.RestoreStockResponse, error) {
	var rawItems []*pbInventory.RestoreItem
	if req != nil {
		rawItems = req.Items
	}
	log.Printf("[InventoryGRPCHandler.RestoreStock] Incoming gRPC restore request: %d items", len(rawItems))

	items, err := parseStockItems(rawItems, func(i *pbInventory.RestoreItem) (string, int32, bool) {
		if i == nil {
			return "", 0, false
		}
		return i.TicketTypeId, i.Quantity, true
	})
	if err != nil {
		log.Printf("[InventoryGRPCHandler.RestoreStock] Validation failed: %v", err)
		return nil, err
	}

	if err := h.service.RestoreStock(ctx, dto.RestoreStockRequest{Items: items}); err != nil {
		log.Printf("[InventoryGRPCHandler.RestoreStock] Service error restoring stock: %v", err)
		return nil, err
	}

	log.Printf("[InventoryGRPCHandler.RestoreStock] Successfully restored stock for %d items", len(items))
	return &pbInventory.RestoreStockResponse{
		Success: true,
	}, nil
}

func parseStockItems[T any](rawItems []T, extract func(T) (string, int32, bool)) ([]dto.DeductStockItemDTO, error) {
	if len(rawItems) == 0 {
		return nil, grpcErrors.InvalidArgument("items cannot be empty")
	}

	items := make([]dto.DeductStockItemDTO, 0, len(rawItems))
	for _, raw := range rawItems {
		ticketTypeIDStr, qty, ok := extract(raw)
		if !ok {
			return nil, grpcErrors.InvalidArgument("item cannot be nil")
		}
		if qty <= 0 {
			return nil, grpcErrors.InvalidArgument("quantity must be greater than 0 for ticket_type_id: %s", ticketTypeIDStr)
		}
		ticketTypeID, err := uuid.Parse(ticketTypeIDStr)
		if err != nil || ticketTypeID == uuid.Nil {
			return nil, grpcErrors.InvalidArgument("invalid ticket_type_id format: %s", ticketTypeIDStr)
		}
		items = append(items, dto.DeductStockItemDTO{
			TicketTypeID: ticketTypeID,
			Quantity:     qty,
		})
	}
	return items, nil
}
