package services_test

import (
	"context"
	"testing"
	"time"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/cache"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
)

// mockDBTX implements db.DBTX for testing
type mockDBTX struct {
	queryRowFunc func(ctx context.Context, query string, args ...interface{}) pgx.Row
	queryFunc    func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	execFunc     func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *mockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return &mockRow{}
}

func (m *mockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return &mockRows{}, nil
}

func (m *mockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return pgconn.NewCommandTag(""), nil
}

type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (r *mockRow) Scan(dest ...interface{}) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return nil
}

type mockRows struct {
	rows [][]interface{}
	idx  int
	err  error
}

func (r *mockRows) Close() {}
func (r *mockRows) Err() error { return r.err }
func (r *mockRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("") }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) TypeMap() *pgtype.Map { return pgtype.NewMap() }
func (r *mockRows) Next() bool {
	if r.idx < len(r.rows) {
		r.idx++
		return true
	}
	return false
}
func (r *mockRows) Scan(dest ...interface{}) error {
	curr := r.rows[r.idx-1]
	for i, d := range dest {
		switch target := d.(type) {
		case *uuid.UUID:
			*target = curr[i].(uuid.UUID)
		case *int32:
			*target = curr[i].(int32)
		case **uuid.UUID:
			if curr[i] != nil {
				val := curr[i].(uuid.UUID)
				*target = &val
			} else {
				*target = nil
			}
		case *time.Time:
			*target = curr[i].(time.Time)
		case **time.Time:
			if curr[i] != nil {
				val := curr[i].(time.Time)
				*target = &val
			} else {
				*target = nil
			}
		}
	}
	return nil
}
func (r *mockRows) Values() ([]interface{}, error) { return nil, nil }
func (r *mockRows) RawValues() [][]byte { return nil }
func (r *mockRows) Conn() *pgx.Conn { return nil }

type testRepo struct {
	dbtx *mockDBTX
}

func (r *testRepo) Queries() *db.Queries {
	return db.New(r.dbtx)
}

func (r *testRepo) GetPool() *pgxpool.Pool {
	return nil
}

func (r *testRepo) ExecTx(ctx context.Context, fn func(q *db.Queries) error) error {
	return fn(db.New(r.dbtx))
}

func TestCreateInventoryBatch_Validations(t *testing.T) {
	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}}, nil)
	ctx := context.Background()

	// Test empty items
	_, err := svc.CreateInventoryBatch(ctx, nil, dto.CreateInventoryBatchRequest{
		Items: []dto.CreateInventoryItem{},
	})
	if err != dto.ErrEmptyBatch {
		t.Fatalf("expected ErrEmptyBatch, got %v", err)
	}

	// Test duplicate items in batch
	sameID := uuid.New()
	_, err = svc.CreateInventoryBatch(ctx, nil, dto.CreateInventoryBatchRequest{
		Items: []dto.CreateInventoryItem{
			{TicketTypeID: sameID, TotalQuantity: 10},
			{TicketTypeID: sameID, TotalQuantity: 20},
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate error, got nil")
	}
}

func TestCreateInventoryBatch_ConflictExists(t *testing.T) {
	ticketID := uuid.New()
	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// Simulate existing ticket id found
			return &mockRows{
				rows: [][]interface{}{
					{ticketID},
				},
			}, nil
		},
	}
	svc := services.NewInventoryService(&testRepo{dbtx: mock}, nil)
	ctx := context.Background()

	_, err := svc.CreateInventoryBatch(ctx, nil, dto.CreateInventoryBatchRequest{
		Items: []dto.CreateInventoryItem{
			{TicketTypeID: ticketID, TotalQuantity: 100},
		},
	})

	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
}

func TestCreateInventoryBatch_Success(t *testing.T) {
	ticketID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()

	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// No existing ticket IDs
			return &mockRows{rows: [][]interface{}{}}, nil
		},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					// Check whether scanning transaction, inventory, or item
					if len(dest) == 7 { // CreateInventoryTransaction
						*dest[0].(*uuid.UUID) = uuid.New()
						*dest[1].(*string) = "INITIAL_STOCK"
						*dest[2].(**string) = nil
						*dest[3].(**uuid.UUID) = nil
						*dest[4].(**string) = nil
						*dest[5].(**uuid.UUID) = nil
						*dest[6].(*time.Time) = now
					} else if len(dest) == 11 { // CreateTicketInventory
						*dest[0].(*uuid.UUID) = invID
						*dest[1].(*uuid.UUID) = ticketID
						*dest[2].(*int32) = 100
						*dest[3].(*int32) = 0
						*dest[4].(*int32) = 0
						*dest[5].(**uuid.UUID) = nil
						*dest[6].(*time.Time) = now
						*dest[7].(**uuid.UUID) = nil
						*dest[8].(*time.Time) = now
						*dest[9].(**uuid.UUID) = nil
						*dest[10].(**time.Time) = nil
					} else if len(dest) == 5 { // CreateInventoryTransactionItem
						*dest[0].(*uuid.UUID) = uuid.New()
						*dest[1].(*uuid.UUID) = uuid.New()
						*dest[2].(*uuid.UUID) = invID
						*dest[3].(*int32) = 100
						*dest[4].(*time.Time) = now
					}
					return nil
				},
			}
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mock}, nil)
	ctx := context.Background()

	result, err := svc.CreateInventoryBatch(ctx, nil, dto.CreateInventoryBatchRequest{
		Items: []dto.CreateInventoryItem{
			{TicketTypeID: ticketID, TotalQuantity: 100},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].AvailableQuantity != 100 {
		t.Fatalf("expected available quantity 100, got %d", result[0].AvailableQuantity)
	}
}

func TestUpdateInventoryStockBatch_NotFound(t *testing.T) {
	ticketID := uuid.New()
	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// Empty returned rows => inventory not found
			return &mockRows{rows: [][]interface{}{}}, nil
		},
	}
	svc := services.NewInventoryService(&testRepo{dbtx: mock}, nil)
	ctx := context.Background()

	_, err := svc.UpdateInventoryStockBatch(ctx, nil, dto.UpdateInventoryBatchRequest{
		Items: []dto.UpdateInventoryItem{
			{TicketTypeID: ticketID, TotalQuantity: 50},
		},
	})
	if err == nil {
		t.Fatalf("expected not found error, got nil")
	}
}

func TestUpdateInventoryStockBatch_InvalidQuantityConstraint(t *testing.T) {
	ticketID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()

	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// Return inventory with held=10, sold=40 (total held+sold = 50)
			return &mockRows{
				rows: [][]interface{}{
					{invID, ticketID, int32(60), int32(10), int32(40), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
	}
	svc := services.NewInventoryService(&testRepo{dbtx: mock}, nil)
	ctx := context.Background()

	// Try updating to total_quantity = 30 (< 50)
	_, err := svc.UpdateInventoryStockBatch(ctx, nil, dto.UpdateInventoryBatchRequest{
		Items: []dto.UpdateInventoryItem{
			{TicketTypeID: ticketID, TotalQuantity: 30},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid quantity error, got nil")
	}
}

func TestUpdateInventoryStockBatch_Success(t *testing.T) {
	ticketID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()

	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			return &mockRows{
				rows: [][]interface{}{
					{invID, ticketID, int32(60), int32(10), int32(20), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					if len(dest) == 7 { // CreateInventoryTransaction
						*dest[0].(*uuid.UUID) = uuid.New()
						*dest[1].(*string) = "ADJUSTMENT"
						*dest[2].(**string) = nil
						*dest[3].(**uuid.UUID) = nil
						*dest[4].(**string) = nil
						*dest[5].(**uuid.UUID) = nil
						*dest[6].(*time.Time) = now
					} else if len(dest) == 11 { // UpdateTicketInventoryStock
						*dest[0].(*uuid.UUID) = invID
						*dest[1].(*uuid.UUID) = ticketID
						*dest[2].(*int32) = 100 // new total
						*dest[3].(*int32) = 10
						*dest[4].(*int32) = 20
						*dest[5].(**uuid.UUID) = nil
						*dest[6].(*time.Time) = now
						*dest[7].(**uuid.UUID) = nil
						*dest[8].(*time.Time) = now
						*dest[9].(**uuid.UUID) = nil
						*dest[10].(**time.Time) = nil
					} else if len(dest) == 5 { // CreateInventoryTransactionItem
						*dest[0].(*uuid.UUID) = uuid.New()
						*dest[1].(*uuid.UUID) = uuid.New()
						*dest[2].(*uuid.UUID) = invID
						*dest[3].(*int32) = 40 // diff 100 - 60 = 40
						*dest[4].(*time.Time) = now
					}
					return nil
				},
			}
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mock}, nil)
	ctx := context.Background()

	result, err := svc.UpdateInventoryStockBatch(ctx, nil, dto.UpdateInventoryBatchRequest{
		Items: []dto.UpdateInventoryItem{
			{TicketTypeID: ticketID, TotalQuantity: 100},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	// available = 100 - (10 + 20) = 70
	if result[0].AvailableQuantity != 70 {
		t.Fatalf("expected available quantity 70, got %d", result[0].AvailableQuantity)
	}
}

func TestGetCurrentStock(t *testing.T) {
	ticketID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()

	mock := &mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*uuid.UUID) = invID
					*dest[1].(*uuid.UUID) = ticketID
					*dest[2].(*int32) = 100
					*dest[3].(*int32) = 10
					*dest[4].(*int32) = 20
					*dest[5].(**uuid.UUID) = nil
					*dest[6].(*time.Time) = now
					*dest[7].(**uuid.UUID) = nil
					*dest[8].(*time.Time) = now
					*dest[9].(**uuid.UUID) = nil
					*dest[10].(**time.Time) = nil
					return nil
				},
			}
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mock}, nil)
	ctx := context.Background()

	result, err := svc.GetCurrentStock(ctx, ticketID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 100 - (10 + 20) = 70
	if result.AvailableQuantity != 70 {
		t.Fatalf("expected available 70, got %d", result.AvailableQuantity)
	}
	if !result.HasStock {
		t.Fatalf("expected HasStock to be true")
	}
}

func TestGetCurrentStock_WithCacheHitAndInvalidation(t *testing.T) {
	ticketID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()
	dbGetStockQueryCount := 0

	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// For UpdateInventoryStockBatch lock rows
			return &mockRows{
				rows: [][]interface{}{
					{invID, ticketID, int32(100), int32(10), int32(20), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					if len(dest) == 7 { // CreateInventoryTransaction
						*dest[0].(*uuid.UUID) = uuid.New()
						*dest[1].(*string) = "ADJUSTMENT"
						*dest[2].(**string) = nil
						*dest[3].(**uuid.UUID) = nil
						*dest[4].(**string) = nil
						*dest[5].(**uuid.UUID) = nil
						*dest[6].(*time.Time) = now
					} else if len(dest) == 11 {
						// Could be GetInventoryByTicketTypeID or UpdateTicketInventoryStock
						if len(args) == 1 { // GetInventoryByTicketTypeID takes 1 arg ($1: ticketTypeID)
							dbGetStockQueryCount++
							*dest[0].(*uuid.UUID) = invID
							*dest[1].(*uuid.UUID) = ticketID
							*dest[2].(*int32) = 100
							*dest[3].(*int32) = 10
							*dest[4].(*int32) = 20
							*dest[5].(**uuid.UUID) = nil
							*dest[6].(*time.Time) = now
							*dest[7].(**uuid.UUID) = nil
							*dest[8].(*time.Time) = now
							*dest[9].(**uuid.UUID) = nil
							*dest[10].(**time.Time) = nil
						} else { // UpdateTicketInventoryStock takes 3 args ($1: id, $2: qty, $3: user)
							*dest[0].(*uuid.UUID) = invID
							*dest[1].(*uuid.UUID) = ticketID
							*dest[2].(*int32) = 120
							*dest[3].(*int32) = 10
							*dest[4].(*int32) = 20
							*dest[5].(**uuid.UUID) = nil
							*dest[6].(*time.Time) = now
							*dest[7].(**uuid.UUID) = nil
							*dest[8].(*time.Time) = now
							*dest[9].(**uuid.UUID) = nil
							*dest[10].(**time.Time) = nil
						}
					} else if len(dest) == 5 { // CreateInventoryTransactionItem
						*dest[0].(*uuid.UUID) = uuid.New()
						*dest[1].(*uuid.UUID) = uuid.New()
						*dest[2].(*uuid.UUID) = invID
						*dest[3].(*int32) = 20
						*dest[4].(*time.Time) = now
					}
					return nil
				},
			}
		},
	}

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := services.NewInventoryService(&testRepo{dbtx: mock}, memCache)
	ctx := context.Background()

	// 1. First Call: Cache Miss -> Queries DB (dbGetStockQueryCount = 1)
	res1, err := svc.GetCurrentStock(ctx, ticketID)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if res1.AvailableQuantity != 70 || !res1.HasStock {
		t.Fatalf("expected available quantity 70 and has_stock true, got %+v", res1)
	}
	if dbGetStockQueryCount != 1 {
		t.Fatalf("expected 1 DB query on cache miss, got %d", dbGetStockQueryCount)
	}

	// 2. Second Call: Cache Hit -> O(1) integer retrieval from cache, NO DB call (dbGetStockQueryCount remains 1)
	res2, err := svc.GetCurrentStock(ctx, ticketID)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if res2.AvailableQuantity != 70 || !res2.HasStock {
		t.Fatalf("expected available quantity 70 from cache, got %+v", res2)
	}
	if dbGetStockQueryCount != 1 {
		t.Fatalf("expected DB query count to remain 1 due to cache hit, got %d", dbGetStockQueryCount)
	}

	// 3. Update Inventory Stock -> Must invalidate cache!
	_, err = svc.UpdateInventoryStockBatch(ctx, nil, dto.UpdateInventoryBatchRequest{
		Items: []dto.UpdateInventoryItem{
			{TicketTypeID: ticketID, TotalQuantity: 120},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error on stock update: %v", err)
	}

	// 4. Third Call: Cache should be invalidated -> Queries DB again (dbGetStockQueryCount = 2)
	res3, err := svc.GetCurrentStock(ctx, ticketID)
	if err != nil {
		t.Fatalf("unexpected error on third call: %v", err)
	}
	if res3.AvailableQuantity != 70 {
		t.Fatalf("expected result, got %+v", res3)
	}
	if dbGetStockQueryCount != 2 {
		t.Fatalf("expected DB GetStock count to be 2 after invalidation, got %d", dbGetStockQueryCount)
	}
}

func TestDeductStock_Validation(t *testing.T) {
	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}}, nil)
	ctx := context.Background()

	// 1. Empty items
	err := svc.DeductStock(ctx, dto.DeductStockRequest{Items: []dto.DeductStockItemDTO{}})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on empty items, got %v", err)
	}

	// 2. Nil ticket_type_id
	err = svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: uuid.Nil, Quantity: 1},
		},
	})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on nil UUID, got %v", err)
	}

	// 3. Zero or negative quantity
	validID := uuid.New()
	err = svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: validID, Quantity: 0},
		},
	})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on 0 quantity, got %v", err)
	}
}

func TestDeductStock_MergeDuplicates(t *testing.T) {
	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	ticketID := uuid.New()
	ctx := context.Background()

	// Initialize stock = 10
	_ = memCache.SetInt(ctx, "inventory:available:"+ticketID.String(), 10, 5*time.Minute)

	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}}, memCache)

	// Send duplicate ticketID items: 4 + 3 = 7
	err := svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: ticketID, Quantity: 4},
			{TicketTypeID: ticketID, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error on deduplication: %v", err)
	}

	// Remaining stock should be 10 - 7 = 3
	val, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID.String())
	if val != 3 {
		t.Fatalf("expected stock 3 after merging duplicates, got %d", val)
	}

	// Requesting more than available after merge: 2 + 2 = 4 (> 3)
	err = svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: ticketID, Quantity: 2},
			{TicketTypeID: ticketID, Quantity: 2},
		},
	})
	if ok, id := grpcErrors.IsInsufficientStock(err); !ok || id != ticketID.String() {
		t.Fatalf("expected INSUFFICIENT_STOCK for %s, got %v", ticketID, err)
	}
}

func TestDeductStock_Atomicity_AllOrNothing(t *testing.T) {
	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	ticketID1 := uuid.New()
	ticketID2 := uuid.New()
	ctx := context.Background()

	// ticket1 has 10, ticket2 has 2
	_ = memCache.SetInt(ctx, "inventory:available:"+ticketID1.String(), 10, 5*time.Minute)
	_ = memCache.SetInt(ctx, "inventory:available:"+ticketID2.String(), 2, 5*time.Minute)

	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}}, memCache)

	// Request ticket1: 5 (sufficient), ticket2: 5 (INSUFFICIENT)
	err := svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: ticketID1, Quantity: 5},
			{TicketTypeID: ticketID2, Quantity: 5},
		},
	})

	if ok, id := grpcErrors.IsInsufficientStock(err); !ok || id != ticketID2.String() {
		t.Fatalf("expected INSUFFICIENT_STOCK for ticket2, got %v", err)
	}

	// Atomicity check: ticket1 MUST NOT be deducted! Stock must remain 10
	val1, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID1.String())
	val2, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID2.String())
	if val1 != 10 {
		t.Fatalf("atomicity violated! ticket1 stock changed to %d (expected 10)", val1)
	}
	if val2 != 2 {
		t.Fatalf("ticket2 stock changed to %d (expected 2)", val2)
	}
}

func TestDeductStock_CacheMiss_FallbackToDB(t *testing.T) {
	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	ticketID1 := uuid.New()
	ticketID2 := uuid.New()
	now := time.Now().UTC()
	ctx := context.Background()

	dbBatchQueried := false
	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			dbBatchQueried = true
			return &mockRows{
				rows: [][]interface{}{
					// id, ticket_type_id, total, held, sold, ...
					{uuid.New(), ticketID1, int32(50), int32(5), int32(5), nil, now, nil, now, nil, nil},
					{uuid.New(), ticketID2, int32(30), int32(0), int32(10), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mock}, memCache)

	// Neither key exists in cache yet! Cache miss triggers fallback to DB
	err := svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: ticketID1, Quantity: 10},
			{TicketTypeID: ticketID2, Quantity: 5},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !dbBatchQueried {
		t.Fatalf("expected DB fallback batch query to be executed on cache miss")
	}

	// Available in DB was:
	// ticket1: 50 - (5 + 5) = 40. Deducted 10 -> 30 remaining
	// ticket2: 30 - (0 + 10) = 20. Deducted 5 -> 15 remaining
	v1, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID1.String())
	v2, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID2.String())
	if v1 != 30 {
		t.Fatalf("expected ticket1 stock 30 in cache, got %d", v1)
	}
	if v2 != 15 {
		t.Fatalf("expected ticket2 stock 15 in cache, got %d", v2)
	}
}

func TestDeductStock_CacheMiss_NotFoundInDB(t *testing.T) {
	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	ticketID := uuid.New()
	ctx := context.Background()

	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// DB returns empty rows (not found)
			return &mockRows{rows: [][]interface{}{}}, nil
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mock}, memCache)

	err := svc.DeductStock(ctx, dto.DeductStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: ticketID, Quantity: 1},
		},
	})
	if grpcErrors.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound on DB cache miss, got %v", err)
	}
}

func TestRestoreStock_SuccessAndMergeDuplicates(t *testing.T) {
	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	ticketID1 := uuid.New()
	ticketID2 := uuid.New()
	missingID := uuid.New()
	ctx := context.Background()

	_ = memCache.SetInt(ctx, "inventory:available:"+ticketID1.String(), 5, 5*time.Minute)
	_ = memCache.SetInt(ctx, "inventory:available:"+ticketID2.String(), 10, 5*time.Minute)

	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}}, memCache)

	err := svc.RestoreStock(ctx, dto.RestoreStockRequest{
		Items: []dto.DeductStockItemDTO{
			{TicketTypeID: ticketID1, Quantity: 2},
			{TicketTypeID: ticketID1, Quantity: 3},
			{TicketTypeID: ticketID2, Quantity: 4},
			{TicketTypeID: missingID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error on RestoreStock: %v", err)
	}

	v1, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID1.String())
	v2, _ := memCache.GetInt(ctx, "inventory:available:"+ticketID2.String())
	if v1 != 10 {
		t.Fatalf("expected ticket1 stock 10 after restore, got %d", v1)
	}
	if v2 != 14 {
		t.Fatalf("expected ticket2 stock 14 after restore, got %d", v2)
	}
	if _, err := memCache.GetInt(ctx, "inventory:available:"+missingID.String()); err != cache.ErrCacheMiss {
		t.Fatalf("expected uncached ticket to remain cache miss, got err=%v", err)
	}
}

func TestRestoreStock_Validation(t *testing.T) {
	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}}, nil)
	ctx := context.Background()

	if err := svc.RestoreStock(ctx, dto.RestoreStockRequest{}); grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on empty items, got %v", err)
	}
	if err := svc.RestoreStock(ctx, dto.RestoreStockRequest{
		Items: []dto.DeductStockItemDTO{{TicketTypeID: uuid.Nil, Quantity: 1}},
	}); grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on nil UUID, got %v", err)
	}
	if err := svc.RestoreStock(ctx, dto.RestoreStockRequest{
		Items: []dto.DeductStockItemDTO{{TicketTypeID: uuid.New(), Quantity: 0}},
	}); grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on 0 quantity, got %v", err)
	}
}

func TestHandleOrderCreated_BatchQueries(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()
	ticket1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ticket2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	inv1 := uuid.New()
	inv2 := uuid.New()
	now := time.Now()

	var queryRowCalls, queryCalls, execCalls int
	var batchTicketIDs []uuid.UUID
	var batchQtys []int32
	var insertedInvIDs []uuid.UUID
	var insertedQtys []int32

	mockDB := &mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			queryRowCalls++
			if queryRowCalls == 1 {
				// TryAcquireIdempotencyKey
				return &mockRow{scanFunc: func(dest ...interface{}) error {
					*(dest[0].(*string)) = args[0].(string)
					return nil
				}}
			}
			// CreateInventoryTransaction
			return &mockRow{scanFunc: func(dest ...interface{}) error {
				*(dest[0].(*uuid.UUID)) = args[0].(uuid.UUID)
				*(dest[1].(*string)) = args[1].(string)
				*(dest[6].(*time.Time)) = now
				return nil
			}}
		},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			queryCalls++
			batchTicketIDs = args[0].([]uuid.UUID)
			batchQtys = args[1].([]int32)
			return &mockRows{
				rows: [][]interface{}{
					{inv1, ticket1, int32(100), int32(5), int32(0), nil, now, nil, now, nil, nil},
					{inv2, ticket2, int32(100), int32(2), int32(0), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
			execCalls++
			insertedInvIDs = args[2].([]uuid.UUID)
			insertedQtys = args[3].([]int32)
			return pgconn.NewCommandTag("INSERT 0 2"), nil
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mockDB}, nil)
	err := svc.HandleOrderCreated(context.Background(), "outbox-1", events.OrderCreatedPayload{
		ID:        orderID,
		UserID:    userID,
		OrderCode: "ORD-TEST-01",
		Items: []events.OrderCreatedItem{
			{TicketTypeID: ticket2, Quantity: 2},
			{TicketTypeID: ticket1, Quantity: 3},
			{TicketTypeID: ticket1, Quantity: 2}, // duplicate ticket_type_id should merge to 5
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if queryRowCalls != 2 || queryCalls != 1 || execCalls != 1 {
		t.Fatalf("expected 2 QueryRow + 1 Query + 1 Exec calls (constant 4 queries), got QueryRow=%d Query=%d Exec=%d",
			queryRowCalls, queryCalls, execCalls)
	}
	if len(batchTicketIDs) != 2 || batchTicketIDs[0] != ticket1 || batchTicketIDs[1] != ticket2 {
		t.Fatalf("unexpected sorted ticket IDs: %v", batchTicketIDs)
	}
	if len(batchQtys) != 2 || batchQtys[0] != 5 || batchQtys[1] != 2 {
		t.Fatalf("unexpected merged quantities: %v", batchQtys)
	}
	if len(insertedInvIDs) != 2 || insertedInvIDs[0] != inv1 || insertedInvIDs[1] != inv2 {
		t.Fatalf("unexpected inserted inventory IDs: %v", insertedInvIDs)
	}
	if len(insertedQtys) != 2 || insertedQtys[0] != 5 || insertedQtys[1] != 2 {
		t.Fatalf("unexpected inserted quantities: %v", insertedQtys)
	}
}

func TestHandleOrderCreated_BatchPartialInsufficientStockFails(t *testing.T) {
	orderID := uuid.New()
	ticket1 := uuid.New()
	ticket2 := uuid.New()
	now := time.Now()

	var queryRowCalls int
	mockDB := &mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			queryRowCalls++
			if queryRowCalls == 1 {
				return &mockRow{scanFunc: func(dest ...interface{}) error {
					*(dest[0].(*string)) = args[0].(string)
					return nil
				}}
			}
			return &mockRow{scanFunc: func(dest ...interface{}) error {
				*(dest[0].(*uuid.UUID)) = args[0].(uuid.UUID)
				return nil
			}}
		},
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			// Only 1 of 2 ticket types had sufficient stock
			return &mockRows{
				rows: [][]interface{}{
					{uuid.New(), ticket1, int32(100), int32(1), int32(0), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mockDB}, nil)
	err := svc.HandleOrderCreated(context.Background(), "outbox-2", events.OrderCreatedPayload{
		ID:        orderID,
		UserID:    uuid.New(),
		OrderCode: "ORD-TEST-02",
		Items: []events.OrderCreatedItem{
			{TicketTypeID: ticket1, Quantity: 1},
			{TicketTypeID: ticket2, Quantity: 10},
		},
	})
	if err == nil {
		t.Fatal("expected error when fewer inventories held than requested")
	}
}
