package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/cache"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InventoryService interface {
	CreateInventoryBatch(ctx context.Context, userID *uuid.UUID, req dto.CreateInventoryBatchRequest) ([]dto.InventoryResponse, error)
	UpdateInventoryStockBatch(ctx context.Context, userID *uuid.UUID, req dto.UpdateInventoryBatchRequest) ([]dto.StockResponse, error)
	GetCurrentStock(ctx context.Context, ticketTypeID uuid.UUID) (*dto.StockResponse, error)
	DeductStock(ctx context.Context, req dto.DeductStockRequest) error
	RestoreStock(ctx context.Context, req dto.RestoreStockRequest) error
	HandleOrderCreated(ctx context.Context, eventID string, payload events.OrderCreatedPayload) error
}

type inventoryService struct {
	repo  repository.InventoryRepository
	cache cache.Cache
}

func stockAvailableCacheKey(ticketTypeID uuid.UUID) string {
	return fmt.Sprintf("inventory:available:%s", ticketTypeID)
}

func NewInventoryService(repo repository.InventoryRepository, cache cache.Cache) InventoryService {
	return &inventoryService{
		repo:  repo,
		cache: cache,
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

	// Invalidate cache for created ticket types
	if s.cache != nil {
		keys := make([]string, 0, len(req.Items))
		for _, item := range req.Items {
			keys = append(keys, stockAvailableCacheKey(item.TicketTypeID))
		}
		_ = s.cache.Delete(ctx, keys...)
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

	// Invalidate cache for updated ticket types
	if s.cache != nil {
		keys := make([]string, 0, len(req.Items))
		for _, item := range req.Items {
			keys = append(keys, stockAvailableCacheKey(item.TicketTypeID))
		}
		_ = s.cache.Delete(ctx, keys...)
	}

	return updatedStocks, nil
}

// GetCurrentStock checks available stock for a single ticketTypeID using Redis Integer Counter
// with Distributed Mutex Lock (PRO_LOCK_KEY_ITEM + ticketId) and Retry to prevent Cache Breakdown.
func (s *inventoryService) GetCurrentStock(ctx context.Context, ticketTypeID uuid.UUID) (*dto.StockResponse, error) {
	cacheKey := stockAvailableCacheKey(ticketTypeID)
	lockKey := cache.ProLockKeyItem(ticketTypeID)

	available, err := cache.GetIntWithLock(
		ctx,
		s.cache,
		cacheKey,
		lockKey,
		5*time.Minute,
		func(ctx context.Context) (int64, error) {
			inv, err := s.repo.Queries().GetInventoryByTicketTypeID(ctx, ticketTypeID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					//cachinng -1 to indicate not found
					return -1, nil
				}
				return 0, fmt.Errorf("get inventory by ticket type id %s: %w", ticketTypeID, err)
			}
			avail := inv.TotalQuantity - (inv.HeldQuantity + inv.SoldQuantity)
			return int64(avail), nil
		},
		cache.LockOptions{
			LockTTL:      5 * time.Second,
			MaxRetries:   10,
			RetryBackoff: 50 * time.Millisecond,
		},
	)
	if err != nil {
		return nil, err
	}
	if available < 0 {
		return nil, dto.ErrInventoryNotFound
	}

	return &dto.StockResponse{
		TicketTypeID:      ticketTypeID,
		AvailableQuantity: int32(available),
		HasStock:          available > 0,
	}, nil
}

// DeductStock atomically checks and deducts stock on Redis across multiple items.
// If cache miss occurs, falls back to DB in a single batch query (no N+1 query),
// populates Redis using SetIntNX (preventing overwrites of concurrent updates),
// and retries the atomic check-and-decrement Lua script.
func (s *inventoryService) DeductStock(ctx context.Context, req dto.DeductStockRequest) error {
	log.Printf("[InventoryService.DeductStock] Received deduct request: %d items", len(req.Items))
	deductItems, err := prepareStockItems(req.Items)
	if err != nil {
		log.Printf("[InventoryService.DeductStock] Validation failed: %v", err)
		return err
	}

	log.Printf("[InventoryService.DeductStock] Starting Redis Lua atomic deduction for items=%+v", deductItems)

	// Step 2 & 3: Redis Transaction (Lua script) with DB Fallback (Step 4)
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		log.Printf("[InventoryService.DeductStock] Attempt %d/%d Redis Lua deduction", attempt+1, maxRetries)
		res, err := s.cache.DeductStockAtomic(ctx, deductItems)
		if err != nil {
			log.Printf("[InventoryService.DeductStock] Redis Lua execution error: %v", err)
			return grpcErrors.Internal("cache deduct stock error: %v", err)
		}

		switch res.Status {
		case cache.DeductStatusOK:
			log.Printf("[InventoryService.DeductStock] Successfully deducted stock atomically on Redis")
			return nil

		case cache.DeductStatusInsufficient:
			failedID := strings.TrimPrefix(res.FailedKey, "inventory:available:")
			log.Printf("[InventoryService.DeductStock] Insufficient stock on Redis for ticket_type_id=%s", failedID)
			return grpcErrors.InsufficientStock(failedID)

		case cache.DeductStatusMissing:
			log.Printf("[InventoryService.DeductStock] Cache miss detected for keys=%v, falling back to database", res.MissingKeys)
			if err := s.initializeMissingStock(ctx, res.MissingKeys); err != nil {
				log.Printf("[InventoryService.DeductStock] Failed to initialize missing stock from DB: %v", err)
				return err
			}
			log.Printf("[InventoryService.DeductStock] Successfully initialized missing keys in Redis, retrying deduction")
			continue
		}
	}

	log.Printf("[InventoryService.DeductStock] Failed to deduct stock after %d retries", maxRetries)
	return grpcErrors.Internal("failed to deduct stock after retries")
}

// RestoreStock atomically restores previously deducted stock on Redis across multiple items.
// Keys that are no longer cached in Redis are skipped so stale partial counts are not created.
func (s *inventoryService) RestoreStock(ctx context.Context, req dto.RestoreStockRequest) error {
	log.Printf("[InventoryService.RestoreStock] Received restore request: %d items", len(req.Items))
	restoreItems, err := prepareStockItems(req.Items)
	if err != nil {
		log.Printf("[InventoryService.RestoreStock] Validation failed: %v", err)
		return err
	}
	if s.cache == nil {
		return grpcErrors.Internal("cache is not initialized")
	}

	if err := s.cache.RestoreStockAtomic(ctx, restoreItems); err != nil {
		log.Printf("[InventoryService.RestoreStock] Redis Lua execution error: %v", err)
		return grpcErrors.Internal("cache restore stock error: %v", err)
	}

	log.Printf("[InventoryService.RestoreStock] Successfully restored stock atomically on Redis for %d items", len(restoreItems))
	return nil
}

// HandleOrderCreated idempotently deducts (holds) stock in PostgreSQL for an OrderCreated event
// and records an inventory_transactions ('HOLD') audit trail.
func (s *inventoryService) HandleOrderCreated(ctx context.Context, eventID string, payload events.OrderCreatedPayload) error {
	if payload.ID == uuid.Nil {
		return fmt.Errorf("invalid order_id in OrderCreated event")
	}
	if len(payload.Items) == 0 {
		return nil
	}

	aggregated := make(map[uuid.UUID]int32)
	for _, item := range payload.Items {
		if item.TicketTypeID == uuid.Nil || item.Quantity <= 0 {
			return fmt.Errorf("invalid item in OrderCreated order_id=%s: ticket_type_id=%s qty=%d",
				payload.ID, item.TicketTypeID, item.Quantity)
		}
		aggregated[item.TicketTypeID] += item.Quantity
	}

	sortedIDs := make([]uuid.UUID, 0, len(aggregated))
	for id := range aggregated {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return sortedIDs[i].String() < sortedIDs[j].String()
	})

	idempotencyToken := fmt.Sprintf("OrderCreated:%s", payload.ID.String())
	if len(idempotencyToken) > 64 {
		idempotencyToken = idempotencyToken[:64]
	}

	return s.repo.ExecTx(ctx, func(q *db.Queries) error {
		_, err := q.TryAcquireIdempotencyKey(ctx, db.TryAcquireIdempotencyKeyParams{
			Token:     idempotencyToken,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				log.Printf("[InventoryService.HandleOrderCreated] Duplicate event skipped idempotently: token=%s event_id=%s", idempotencyToken, eventID)
				return nil
			}
			return fmt.Errorf("acquire idempotency key %s: %w", idempotencyToken, err)
		}

		refType := "ORDER"
		note := fmt.Sprintf("Hold stock for OrderCreated %s (event_id=%s)", payload.OrderCode, eventID)
		txRecord, err := q.CreateInventoryTransaction(ctx, db.CreateInventoryTransactionParams{
			ID:              uuid.New(),
			TransactionType: "HOLD",
			ReferenceType:   &refType,
			ReferenceID:     &payload.ID,
			Note:            &note,
			CreatedBy:       &payload.UserID,
		})
		if err != nil {
			return fmt.Errorf("create HOLD inventory transaction: %w", err)
		}

		sortedQtys := make([]int32, len(sortedIDs))
		for i, ticketTypeID := range sortedIDs {
			sortedQtys[i] = aggregated[ticketTypeID]
		}

		heldInventories, err := q.HoldTicketInventoryStockBatch(ctx, db.HoldTicketInventoryStockBatchParams{
			TicketTypeIDs: sortedIDs,
			Quantities:    sortedQtys,
		})
		if err != nil {
			return fmt.Errorf("batch hold stock for order_id=%s: %w", payload.ID, err)
		}
		if len(heldInventories) != len(sortedIDs) {
			return fmt.Errorf("insufficient DB stock or missing inventory for order_id=%s (held=%d expected=%d)",
				payload.ID, len(heldInventories), len(sortedIDs))
		}

		itemIDs := make([]uuid.UUID, len(heldInventories))
		inventoryIDs := make([]uuid.UUID, len(heldInventories))
		itemQuantities := make([]int32, len(heldInventories))
		for i, inv := range heldInventories {
			itemIDs[i] = uuid.New()
			inventoryIDs[i] = inv.ID
			itemQuantities[i] = aggregated[inv.TicketTypeID]
		}

		if err := q.CreateInventoryTransactionItemBatch(ctx, db.CreateInventoryTransactionItemBatchParams{
			IDs:                    itemIDs,
			InventoryTransactionID: txRecord.ID,
			TicketInventoryIDs:     inventoryIDs,
			Quantities:             itemQuantities,
		}); err != nil {
			return fmt.Errorf("batch create HOLD transaction items for order_id=%s: %w", payload.ID, err)
		}

		return nil
	})
}

func prepareStockItems(items []dto.DeductStockItemDTO) ([]cache.DeductStockItem, error) {
	if len(items) == 0 {
		return nil, grpcErrors.InvalidArgument("items cannot be empty")
	}

	aggregated := make(map[uuid.UUID]int64)
	for _, item := range items {
		if item.TicketTypeID == uuid.Nil {
			return nil, grpcErrors.InvalidArgument("ticket_type_id must be a valid non-nil UUID")
		}
		if item.Quantity <= 0 {
			return nil, grpcErrors.InvalidArgument("quantity must be greater than 0 for ticket_type_id: %s", item.TicketTypeID)
		}
		aggregated[item.TicketTypeID] += int64(item.Quantity)
	}

	// Deterministic sorting of ticketTypeIDs to prevent any lock order inversion
	sortedIDs := make([]uuid.UUID, 0, len(aggregated))
	for id := range aggregated {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return sortedIDs[i].String() < sortedIDs[j].String()
	})

	result := make([]cache.DeductStockItem, len(sortedIDs))
	for i, id := range sortedIDs {
		result[i] = cache.DeductStockItem{
			Key:      stockAvailableCacheKey(id),
			Quantity: aggregated[id],
		}
	}
	return result, nil
}

// initializeMissingStock queries DB in a single batch query (avoiding N+1 query)
// and initializes Redis with SetIntNX to prevent overwriting newer concurrent stock updates.
func (s *inventoryService) initializeMissingStock(ctx context.Context, missingKeys []string) error {
	log.Printf("[InventoryService.initializeMissingStock] Processing missing keys: %v", missingKeys)
	missingIDs := make([]uuid.UUID, 0, len(missingKeys))
	for _, k := range missingKeys {
		idStr := strings.TrimPrefix(k, "inventory:available:")
		parsed, err := uuid.Parse(idStr)
		if err == nil {
			missingIDs = append(missingIDs, parsed)
		}
	}

	if len(missingIDs) == 0 {
		log.Printf("[InventoryService.initializeMissingStock] Error: failed to parse any missing keys")
		return grpcErrors.Internal("failed to parse missing cache keys")
	}

	// Sort missing IDs to acquire distributed locks in deterministic order
	sort.Slice(missingIDs, func(i, j int) bool {
		return missingIDs[i].String() < missingIDs[j].String()
	})

	// Acquire distributed locks to prevent Cache Stampede
	var lockedKeys []string
	for _, id := range missingIDs {
		lockKey := cache.ProLockKeyItem(id)
		acquired, lockErr := s.cache.AcquireLock(ctx, lockKey, 5*time.Second)
		if lockErr == nil && acquired {
			lockedKeys = append(lockedKeys, lockKey)
		}
	}
	defer func() {
		for _, lk := range lockedKeys {
			_ = s.cache.ReleaseLock(ctx, lk)
		}
	}()

	// Double-check if keys became present while acquiring locks
	stillMissing := make([]uuid.UUID, 0, len(missingIDs))
	for _, id := range missingIDs {
		exists, _ := s.cache.Exists(ctx, stockAvailableCacheKey(id))
		if !exists {
			stillMissing = append(stillMissing, id)
		}
	}

	if len(stillMissing) == 0 {
		log.Printf("[InventoryService.initializeMissingStock] Keys were already populated by concurrent request")
		return nil
	}

	log.Printf("[InventoryService.initializeMissingStock] Querying DB for %d still missing IDs: %v", len(stillMissing), stillMissing)
	// Batch query Database for all still-missing IDs in ONE single query (avoid N+1 query!)
	inventories, err := s.repo.Queries().GetInventoriesByTicketTypeIDs(ctx, stillMissing)
	if err != nil {
		log.Printf("[InventoryService.initializeMissingStock] DB batch query error: %v", err)
		return grpcErrors.Internal("database query inventory error: %v", err)
	}

	foundMap := make(map[uuid.UUID]db.TicketInventory, len(inventories))
	for _, inv := range inventories {
		foundMap[inv.TicketTypeID] = inv
	}

	// Check if any ticket_type_id was not found in DB
	for _, id := range stillMissing {
		if _, found := foundMap[id]; !found {
			log.Printf("[InventoryService.initializeMissingStock] Ticket type %s not found in database", id)
			return grpcErrors.NotFound("inventory for ticket_type_id", id)
		}
	}

	// Synchronize stock from Database to Redis using SetIntNX
	for _, inv := range inventories {
		available := int64(inv.TotalQuantity - (inv.HeldQuantity + inv.SoldQuantity))
		if available < 0 {
			available = 0
		}
		cacheKey := stockAvailableCacheKey(inv.TicketTypeID)
		// SetIntNX ensures we never overwrite newer stock updated by another concurrent request
		_, _ = s.cache.SetIntNX(ctx, cacheKey, available, 0)
		log.Printf("[InventoryService.initializeMissingStock] Synchronized stock to Redis: key=%s, available=%d", cacheKey, available)
	}

	return nil
}
