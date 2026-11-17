package services

import (
	"context"
	"fmt"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/repository"
	"github.com/google/uuid"
)

type InventoryService interface {
	CreateInventoryBatch(ctx context.Context, userID *uuid.UUID, req dto.CreateInventoryBatchRequest) ([]dto.InventoryResponse, error)
	UpdateInventoryStockBatch(ctx context.Context, userID *uuid.UUID, req dto.UpdateInventoryBatchRequest) ([]dto.StockResponse, error)
	GetCurrentStock(ctx context.Context, ticketTypeIDs []uuid.UUID) ([]dto.StockResponse, error)
}

type inventoryService struct {
	repo repository.InventoryRepository
}

func NewInventoryService(repo repository.InventoryRepository) InventoryService {
	return &inventoryService{
		repo: repo,
	}
}

func (s *inventoryService) CreateInventoryBatch(ctx context.Context, userID *uuid.UUID, req dto.CreateInventoryBatchRequest) ([]dto.InventoryResponse, error) {
	if len(req.Items) == 0 {
		return nil, dto.ErrEmptyBatch
	}

	seen := make(map[uuid.UUID]bool, len(req.Items))
	ticketTypeIDs := make([]uuid.UUID, 0, len(req.Items))
	for _, item := range req.Items {
		if seen[item.TicketTypeID] {
			return nil, fmt.Errorf("%w: %s", dto.ErrDuplicateTicketTypeID, item.TicketTypeID)
		}
		seen[item.TicketTypeID] = true
		ticketTypeIDs = append(ticketTypeIDs, item.TicketTypeID)
	}

	var createdInventories []dto.InventoryResponse

	err := s.repo.ExecTx(ctx, func(q *db.Queries) error {
		// 1. Check if any ticket_type_id already exists in ticket_inventories
		existing, err := q.GetExistingTicketTypeIDs(ctx, ticketTypeIDs)
		if err != nil {
			return fmt.Errorf("check existing ticket types: %w", err)
		}
		if len(existing) > 0 {
			return fmt.Errorf("%w: ticket_type_id %s", dto.ErrInventoryAlreadyExists, existing[0])
		}

		// 2. Transaction-first: Insert master audit record into inventory_transactions
		txID := uuid.New()
		refType := req.ReferenceType
		if refType == nil {
			defaultType := "INITIAL_IMPORT"
			refType = &defaultType
		}

		txRecord, err := q.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ID:              txID,
			TransactionType: "INITIAL_STOCK",
			ReferenceType:   refType,
			ReferenceID:     req.ReferenceID,
			Note:            req.Note,
			CreatedBy:       userID,
		})
		if err != nil {
			return fmt.Errorf("create inventory transaction: %w", err)
		}

		// 3. Batch insert into ticket_inventories and transaction items
		for _, item := range req.Items {
			invID := uuid.New()
			invRecord, err := q.CreateTicketInventory(ctx, db.CreateTicketInventoryParams{
				ID:            invID,
				TicketTypeID:  item.TicketTypeID,
				TotalQuantity: item.TotalQuantity,
				CreatedBy:     userID,
			})
			if err != nil {
				return fmt.Errorf("create ticket inventory for %s: %w", item.TicketTypeID, err)
			}

			// Insert into inventory_transaction_items if quantity > 0
			if item.TotalQuantity > 0 {
				_, err = q.CreateInventoryTransactionItem(ctx, db.CreateInventoryTransactionItemParams{
					ID:                     uuid.New(),
					InventoryTransactionID: txRecord.ID,
					TicketInventoryID:      invRecord.ID,
					Quantity:               item.TotalQuantity,
				})
				if err != nil {
					return fmt.Errorf("create transaction item for inventory %s: %w", invRecord.ID, err)
				}
			}

			available := invRecord.TotalQuantity - (invRecord.HeldQuantity + invRecord.SoldQuantity)
			createdInventories = append(createdInventories, dto.InventoryResponse{
				ID:                invRecord.ID,
				TicketTypeID:      invRecord.TicketTypeID,
				TotalQuantity:     invRecord.TotalQuantity,
				HeldQuantity:      invRecord.HeldQuantity,
				SoldQuantity:      invRecord.SoldQuantity,
				AvailableQuantity: available,
				CreatedBy:         invRecord.CreatedBy,
				CreatedAt:         invRecord.CreatedAt,
				UpdatedBy:         invRecord.UpdatedBy,
				UpdatedAt:         invRecord.UpdatedAt,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdInventories, nil
}

func (s *inventoryService) UpdateInventoryStockBatch(ctx context.Context, userID *uuid.UUID, req dto.UpdateInventoryBatchRequest) ([]dto.StockResponse, error) {
	if len(req.Items) == 0 {
		return nil, dto.ErrEmptyBatch
	}

	seen := make(map[uuid.UUID]bool, len(req.Items))
	ticketTypeIDs := make([]uuid.UUID, 0, len(req.Items))
	for _, item := range req.Items {
		if seen[item.TicketTypeID] {
			return nil, fmt.Errorf("%w: %s", dto.ErrDuplicateTicketTypeID, item.TicketTypeID)
		}
		seen[item.TicketTypeID] = true
		ticketTypeIDs = append(ticketTypeIDs, item.TicketTypeID)
	}

	var updatedStocks []dto.StockResponse

	err := s.repo.ExecTx(ctx, func(q *db.Queries) error {
		// 1. Lock rows FOR UPDATE
		inventories, err := q.GetInventoriesByTicketTypeIDsForUpdate(ctx, ticketTypeIDs)
		if err != nil {
			return fmt.Errorf("query inventories for update: %w", err)
		}

		invMap := make(map[uuid.UUID]db.TicketInventory, len(inventories))
		for _, inv := range inventories {
			invMap[inv.TicketTypeID] = inv
		}

		// Validate all IDs exist and quantity constraint
		for _, item := range req.Items {
			existing, found := invMap[item.TicketTypeID]
			if !found {
				return fmt.Errorf("%w: ticket_type_id %s", dto.ErrInventoryNotFound, item.TicketTypeID)
			}
			if item.TotalQuantity < (existing.HeldQuantity + existing.SoldQuantity) {
				return fmt.Errorf("%w: total_quantity (%d) < held (%d) + sold (%d) for ticket_type_id %s",
					dto.ErrInvalidQuantity, item.TotalQuantity, existing.HeldQuantity, existing.SoldQuantity, item.TicketTypeID)
			}
		}

		// 2. Transaction-first: Insert into inventory_transactions
		txID := uuid.New()
		refType := req.ReferenceType
		if refType == nil {
			defaultType := "ADMIN_UPDATE"
			refType = &defaultType
		}

		txRecord, err := q.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ID:              txID,
			TransactionType: "ADJUSTMENT",
			ReferenceType:   refType,
			ReferenceID:     req.ReferenceID,
			Note:            req.Note,
			CreatedBy:       userID,
		})
		if err != nil {
			return fmt.Errorf("create adjustment transaction: %w", err)
		}

		// 3. Update stock and insert transaction items for quantity difference
		for _, item := range req.Items {
			existing := invMap[item.TicketTypeID]
			diff := item.TotalQuantity - existing.TotalQuantity

			updatedInv, err := q.UpdateTicketInventoryStock(ctx, db.UpdateTicketInventoryStockParams{
				ID:            existing.ID,
				TotalQuantity: item.TotalQuantity,
				UpdatedBy:     userID,
			})
			if err != nil {
				return fmt.Errorf("update inventory stock for %s: %w", existing.ID, err)
			}

			if diff != 0 {
				qty := diff
				if qty < 0 {
					qty = -qty
				}
				_, err = q.CreateInventoryTransactionItem(ctx, db.CreateInventoryTransactionItemParams{
					ID:                     uuid.New(),
					InventoryTransactionID: txRecord.ID,
					TicketInventoryID:      existing.ID,
					Quantity:               qty,
				})
				if err != nil {
					return fmt.Errorf("create transaction item for inventory %s: %w", existing.ID, err)
				}
			}

			available := updatedInv.TotalQuantity - (updatedInv.HeldQuantity + updatedInv.SoldQuantity)
			updatedStocks = append(updatedStocks, dto.StockResponse{
				TicketTypeID:      updatedInv.TicketTypeID,
				TotalQuantity:     updatedInv.TotalQuantity,
				HeldQuantity:      updatedInv.HeldQuantity,
				SoldQuantity:      updatedInv.SoldQuantity,
				AvailableQuantity: available,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedStocks, nil
}

func (s *inventoryService) GetCurrentStock(ctx context.Context, ticketTypeIDs []uuid.UUID) ([]dto.StockResponse, error) {
	if len(ticketTypeIDs) == 0 {
		return []dto.StockResponse{}, nil
	}

	// De-duplicate query IDs
	seen := make(map[uuid.UUID]bool, len(ticketTypeIDs))
	uniqueIDs := make([]uuid.UUID, 0, len(ticketTypeIDs))
	for _, id := range ticketTypeIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	inventories, err := s.repo.Queries().GetInventoriesByTicketTypeIDs(ctx, uniqueIDs)
	if err != nil {
		return nil, fmt.Errorf("get inventories by ticket type ids: %w", err)
	}

	result := make([]dto.StockResponse, 0, len(inventories))
	for _, inv := range inventories {
		available := inv.TotalQuantity - (inv.HeldQuantity + inv.SoldQuantity)
		result = append(result, dto.StockResponse{
			TicketTypeID:      inv.TicketTypeID,
			TotalQuantity:     inv.TotalQuantity,
			HeldQuantity:      inv.HeldQuantity,
			SoldQuantity:      inv.SoldQuantity,
			AvailableQuantity: available,
		})
	}

	return result, nil
}
