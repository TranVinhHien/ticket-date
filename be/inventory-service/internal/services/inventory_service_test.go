package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	svc := services.NewInventoryService(&testRepo{dbtx: &mockDBTX{}})
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
	svc := services.NewInventoryService(&testRepo{dbtx: mock})
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

	svc := services.NewInventoryService(&testRepo{dbtx: mock})
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
	svc := services.NewInventoryService(&testRepo{dbtx: mock})
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
	svc := services.NewInventoryService(&testRepo{dbtx: mock})
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

	svc := services.NewInventoryService(&testRepo{dbtx: mock})
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
	ticketID1 := uuid.New()
	ticketID2 := uuid.New()
	now := time.Now().UTC()

	mock := &mockDBTX{
		queryFunc: func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
			return &mockRows{
				rows: [][]interface{}{
					{uuid.New(), ticketID1, int32(100), int32(10), int32(20), nil, now, nil, now, nil, nil},
					{uuid.New(), ticketID2, int32(50), int32(0), int32(5), nil, now, nil, now, nil, nil},
				},
			}, nil
		},
	}

	svc := services.NewInventoryService(&testRepo{dbtx: mock})
	ctx := context.Background()

	result, err := svc.GetCurrentStock(ctx, []uuid.UUID{ticketID1, ticketID2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 stock results, got %d", len(result))
	}

	// For ticketID1: 100 - (10 + 20) = 70
	if result[0].AvailableQuantity != 70 {
		t.Fatalf("expected available 70, got %d", result[0].AvailableQuantity)
	}
	// For ticketID2: 50 - (0 + 5) = 45
	if result[1].AvailableQuantity != 45 {
		t.Fatalf("expected available 45, got %d", result[1].AvailableQuantity)
	}
}
