package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	pbEvent "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/event"
	pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type mockInventoryClient struct {
	deductStockFn  func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error)
	restoreStockFn func(ctx context.Context, in *pbInventory.RestoreStockRequest, opts ...grpc.CallOption) (*pbInventory.RestoreStockResponse, error)
}

func (m *mockInventoryClient) DeductStock(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
	if m.deductStockFn != nil {
		return m.deductStockFn(ctx, in, opts...)
	}
	return &pbInventory.DeductStockResponse{Success: true}, nil
}

func (m *mockInventoryClient) RestoreStock(ctx context.Context, in *pbInventory.RestoreStockRequest, opts ...grpc.CallOption) (*pbInventory.RestoreStockResponse, error) {
	if m.restoreStockFn != nil {
		return m.restoreStockFn(ctx, in, opts...)
	}
	return &pbInventory.RestoreStockResponse{Success: true}, nil
}

type mockEventClient struct {
	getTicketTypesFn func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error)
}

func (m *mockEventClient) GetTicketTypes(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
	if m.getTicketTypesFn != nil {
		return m.getTicketTypesFn(ctx, in, opts...)
	}
	return &pbEvent.GetTicketTypesResponse{}, nil
}

type mockOrderRepo struct {
	queriesFn func() *db.Queries
	execTxFn  func(ctx context.Context, fn func(q *db.Queries) error) error
}

func (m *mockOrderRepo) Queries() *db.Queries {
	if m.queriesFn != nil {
		return m.queriesFn()
	}
	return nil
}

func (m *mockOrderRepo) GetPool() *pgxpool.Pool {
	return nil
}

func (m *mockOrderRepo) ExecTx(ctx context.Context, fn func(q *db.Queries) error) error {
	if m.execTxFn != nil {
		return m.execTxFn(ctx, fn)
	}
	return nil
}

func TestCreateOrder_Validation(t *testing.T) {
	validTicketTypeID := uuid.New()
	negativePriceTicketTypeID := uuid.New()

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			var list []*pbEvent.TicketTypeDetail
			for _, id := range in.TicketTypeIds {
				if id == validTicketTypeID.String() {
					list = append(list, &pbEvent.TicketTypeDetail{
						Id:       id,
						Name:     "Standard",
						Sku:      "STD-01",
						Price:    100000,
						Currency: "VND",
					})
				} else if id == negativePriceTicketTypeID.String() {
					list = append(list, &pbEvent.TicketTypeDetail{
						Id:       id,
						Name:     "Invalid Price",
						Sku:      "INV-01",
						Price:    -100,
						Currency: "VND",
					})
				}
			}
			return &pbEvent.GetTicketTypesResponse{TicketTypes: list}, nil
		},
	}

	service := NewOrderService(&mockOrderRepo{}, nil, &mockInventoryClient{}, eventClient)
	ctx := context.Background()
	validUserID := uuid.New()

	tests := []struct {
		name        string
		userID      uuid.UUID
		req         dto.CreateOrderRequest
		expectedErr error
	}{
		{
			name:        "invalid user id (nil)",
			userID:      uuid.Nil,
			req:         dto.CreateOrderRequest{Items: []dto.CreateOrderItemRequest{{TicketTypeID: validTicketTypeID, Quantity: 1}}},
			expectedErr: dto.ErrInvalidUserID,
		},
		{
			name:        "empty items",
			userID:      validUserID,
			req:         dto.CreateOrderRequest{Items: []dto.CreateOrderItemRequest{}},
			expectedErr: dto.ErrEmptyOrderItems,
		},
		{
			name:   "invalid quantity (<= 0)",
			userID: validUserID,
			req: dto.CreateOrderRequest{
				Items: []dto.CreateOrderItemRequest{
					{TicketTypeID: validTicketTypeID, Quantity: 0},
				},
			},
			expectedErr: dto.ErrInvalidQuantity,
		},
		{
			name:   "ticket type not found in event service",
			userID: validUserID,
			req: dto.CreateOrderRequest{
				Items: []dto.CreateOrderItemRequest{
					{TicketTypeID: uuid.New(), Quantity: 1},
				},
			},
			expectedErr: dto.ErrTicketTypeNotFound,
		},
		{
			name:   "invalid unit price (< 0)",
			userID: validUserID,
			req: dto.CreateOrderRequest{
				Items: []dto.CreateOrderItemRequest{
					{TicketTypeID: negativePriceTicketTypeID, Quantity: 2},
				},
			},
			expectedErr: dto.ErrInvalidUnitPrice,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.CreateOrder(ctx, tc.userID, tc.req)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

// mockDBTX implements db.DBTX to intercept queries in unit tests
type mockDBTX struct {
	capturedOutboxPayload string
	capturedEventType     string
	capturedAggregateID   string
	createdOrderID        uuid.UUID
	createdOrderCode      string
	totalAmount           int64
	currency              string
	status                string
	items                 []db.OrderItem
	orders                map[uuid.UUID]db.Order
	idempotencyKeys       map[string]db.OrderIdempotencyKey
}

type mockRow struct {
	scanFn func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.scanFn != nil {
		return m.scanFn(dest...)
	}
	return nil
}

type mockOrderItemsRows struct {
	items []db.OrderItem
	idx   int
}

func (r *mockOrderItemsRows) Close()                                       {}
func (r *mockOrderItemsRows) Err() error                                   { return nil }
func (r *mockOrderItemsRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("") }
func (r *mockOrderItemsRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockOrderItemsRows) TypeMap() *pgtype.Map                         { return pgtype.NewMap() }
func (r *mockOrderItemsRows) Values() ([]any, error)                       { return nil, nil }
func (r *mockOrderItemsRows) RawValues() [][]byte                          { return nil }
func (r *mockOrderItemsRows) Conn() *pgx.Conn                              { return nil }
func (r *mockOrderItemsRows) Next() bool {
	if r.idx < len(r.items) {
		r.idx++
		return true
	}
	return false
}
func (r *mockOrderItemsRows) Scan(dest ...any) error {
	item := r.items[r.idx-1]
	*dest[0].(*uuid.UUID) = item.ID
	*dest[1].(*uuid.UUID) = item.OrderID
	*dest[2].(*uuid.UUID) = item.TicketTypeID
	*dest[3].(*string) = item.TicketTypeName
	*dest[4].(*string) = item.Sku
	*dest[5].(*int32) = item.Quantity
	*dest[6].(*int64) = item.UnitPrice
	*dest[7].(*int64) = item.Subtotal
	*dest[8].(**uuid.UUID) = item.CreatedBy
	*dest[9].(*time.Time) = item.CreatedAt
	*dest[10].(**uuid.UUID) = item.UpdatedBy
	*dest[11].(*time.Time) = item.UpdatedAt
	*dest[12].(**uuid.UUID) = item.DeletedBy
	*dest[13].(**time.Time) = item.DeletedAt
	return nil
}

func (m *mockDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "DELETE FROM order_idempotency_keys") {
		userID := args[0].(uuid.UUID)
		key := args[1].(string)
		orderID := args[2].(uuid.UUID)
		mapKey := userID.String() + ":" + key
		if m.idempotencyKeys != nil {
			if rec, exists := m.idempotencyKeys[mapKey]; exists && rec.OrderID == orderID {
				delete(m.idempotencyKeys, mapKey)
			}
		}
		return pgconn.NewCommandTag("DELETE 1"), nil
	}
	return pgconn.NewCommandTag(""), nil
}

func (m *mockDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if strings.Contains(query, "FROM order_items") {
		orderID := args[0].(uuid.UUID)
		var matched []db.OrderItem
		for _, item := range m.items {
			if item.OrderID == orderID {
				matched = append(matched, item)
			}
		}
		return &mockOrderItemsRows{items: matched}, nil
	}
	return &mockOrderItemsRows{}, nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	// 1. CreateOrder
	if strings.Contains(query, "INSERT INTO orders") {
		orderID := args[0].(uuid.UUID)
		userID := args[1].(uuid.UUID)
		orderCode := args[2].(string)
		totalAmount := args[3].(int64)
		currency := args[4].(string)
		status := args[5].(string)
		expiresAt := args[6].(*time.Time)
		paidAt := args[7].(*time.Time)
		createdBy := args[8].(*uuid.UUID)

		m.createdOrderID = orderID
		m.createdOrderCode = orderCode
		m.totalAmount = totalAmount
		m.currency = currency
		m.status = status
		now := time.Now()
		if m.orders == nil {
			m.orders = make(map[uuid.UUID]db.Order)
		}
		m.orders[orderID] = db.Order{
			ID:          orderID,
			UserID:      userID,
			OrderCode:   orderCode,
			TotalAmount: totalAmount,
			Currency:    currency,
			Status:      status,
			ExpiresAt:   expiresAt,
			PaidAt:      paidAt,
			CreatedBy:   createdBy,
			CreatedAt:   now,
			UpdatedBy:   createdBy,
			UpdatedAt:   now,
		}

		return &mockRow{
			scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = orderID
				*dest[1].(*uuid.UUID) = userID
				*dest[2].(*string) = orderCode
				*dest[3].(*int64) = totalAmount
				*dest[4].(*string) = currency
				*dest[5].(*string) = status
				*dest[6].(**time.Time) = expiresAt
				*dest[7].(**time.Time) = paidAt
				*dest[8].(**uuid.UUID) = createdBy
				*dest[9].(*time.Time) = now
				*dest[10].(**uuid.UUID) = createdBy
				*dest[11].(*time.Time) = now
				*dest[12].(**uuid.UUID) = nil
				*dest[13].(**time.Time) = nil
				return nil
			},
		}
	}

	// 2. CreateOrderItem
	if strings.Contains(query, "INSERT INTO order_items") {
		itemID := args[0].(uuid.UUID)
		orderID := args[1].(uuid.UUID)
		ticketTypeID := args[2].(uuid.UUID)
		ticketTypeName := args[3].(string)
		sku := args[4].(string)
		quantity := args[5].(int32)
		unitPrice := args[6].(int64)
		subtotal := args[7].(int64)

		item := db.OrderItem{
			ID:             itemID,
			OrderID:        orderID,
			TicketTypeID:   ticketTypeID,
			TicketTypeName: ticketTypeName,
			Sku:            sku,
			Quantity:       quantity,
			UnitPrice:      unitPrice,
			Subtotal:       subtotal,
			CreatedAt:      time.Now(),
		}
		m.items = append(m.items, item)

		return &mockRow{
			scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = itemID
				*dest[1].(*uuid.UUID) = orderID
				*dest[2].(*uuid.UUID) = ticketTypeID
				*dest[3].(*string) = ticketTypeName
				*dest[4].(*string) = sku
				*dest[5].(*int32) = quantity
				*dest[6].(*int64) = unitPrice
				*dest[7].(*int64) = subtotal
				*dest[8].(**uuid.UUID) = nil
				*dest[9].(*time.Time) = item.CreatedAt
				*dest[10].(**uuid.UUID) = nil
				*dest[11].(*time.Time) = item.CreatedAt
				*dest[12].(**uuid.UUID) = nil
				*dest[13].(**time.Time) = nil
				return nil
			},
		}
	}

	// 3. CreateOutboxEvent
	if strings.Contains(query, "INSERT INTO outbox_event") {
		aggID := args[0].(string)
		eventType := args[1].(string)
		payload := args[2].(string)
		status := args[3].(int16)

		m.capturedAggregateID = aggID
		m.capturedEventType = eventType
		m.capturedOutboxPayload = payload

		return &mockRow{
			scanFn: func(dest ...any) error {
				*dest[0].(*int64) = 1
				*dest[1].(*string) = aggID
				*dest[2].(*string) = eventType
				*dest[3].(*string) = payload
				*dest[4].(*int16) = status
				*dest[5].(*time.Time) = time.Now()
				*dest[6].(**time.Time) = nil
				return nil
			},
		}
	}

	// 4. AcquireOrderIdempotencyKey
	if strings.Contains(query, "INSERT INTO order_idempotency_keys") {
		userID := args[0].(uuid.UUID)
		key := args[1].(string)
		orderID := args[2].(uuid.UUID)
		expiresAt := args[3].(time.Time)
		if m.idempotencyKeys == nil {
			m.idempotencyKeys = make(map[string]db.OrderIdempotencyKey)
		}
		mapKey := userID.String() + ":" + key
		if existing, exists := m.idempotencyKeys[mapKey]; exists && existing.ExpiresAt.After(time.Now()) {
			return &mockRow{
				scanFn: func(dest ...any) error {
					*dest[0].(*uuid.UUID) = existing.UserID
					*dest[1].(*string) = existing.Key
					*dest[2].(*uuid.UUID) = existing.OrderID
					*dest[3].(*time.Time) = existing.CreatedAt
					*dest[4].(*time.Time) = existing.ExpiresAt
					*dest[5].(*bool) = false
					return nil
				},
			}
		}
		rec := db.OrderIdempotencyKey{
			UserID:    userID,
			Key:       key,
			OrderID:   orderID,
			CreatedAt: time.Now(),
			ExpiresAt: expiresAt,
		}
		m.idempotencyKeys[mapKey] = rec
		return &mockRow{
			scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = rec.UserID
				*dest[1].(*string) = rec.Key
				*dest[2].(*uuid.UUID) = rec.OrderID
				*dest[3].(*time.Time) = rec.CreatedAt
				*dest[4].(*time.Time) = rec.ExpiresAt
				*dest[5].(*bool) = true
				return nil
			},
		}
	}

	// 5. GetOrderByID
	if strings.Contains(query, "FROM orders") {
		orderID := args[0].(uuid.UUID)
		o, exists := m.orders[orderID]
		if !exists {
			return &mockRow{
				scanFn: func(dest ...any) error {
					return pgx.ErrNoRows
				},
			}
		}
		return &mockRow{
			scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = o.ID
				*dest[1].(*uuid.UUID) = o.UserID
				*dest[2].(*string) = o.OrderCode
				*dest[3].(*int64) = o.TotalAmount
				*dest[4].(*string) = o.Currency
				*dest[5].(*string) = o.Status
				*dest[6].(**time.Time) = o.ExpiresAt
				*dest[7].(**time.Time) = o.PaidAt
				*dest[8].(**uuid.UUID) = o.CreatedBy
				*dest[9].(*time.Time) = o.CreatedAt
				*dest[10].(**uuid.UUID) = o.UpdatedBy
				*dest[11].(*time.Time) = o.UpdatedAt
				*dest[12].(**uuid.UUID) = o.DeletedBy
				*dest[13].(**time.Time) = o.DeletedAt
				return nil
			},
		}
	}

	return &mockRow{}
}

func TestCreateOrder_Success(t *testing.T) {
	mockDB := &mockDBTX{}
	queries := db.New(mockDB)

	repo := &mockOrderRepo{
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			return fn(queries)
		},
	}

	ticketTypeID1 := uuid.New()
	ticketTypeID2 := uuid.New()

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{
						Id:       ticketTypeID1.String(),
						Name:     "VIP Ticket",
						Sku:      "VIP-001",
						Price:    500000,
						Currency: "VND",
					},
					{
						Id:       ticketTypeID2.String(),
						Name:     "Standard Ticket",
						Sku:      "STD-001",
						Price:    200000,
						Currency: "VND",
					},
				},
			}, nil
		},
	}

	service := NewOrderService(repo, nil, &mockInventoryClient{}, eventClient)
	ctx := context.Background()
	userID := uuid.New()

	req := dto.CreateOrderRequest{
		Currency: "VND",
		Items: []dto.CreateOrderItemRequest{
			{
				TicketTypeID: ticketTypeID1,
				Quantity:     2,
			},
			{
				TicketTypeID: ticketTypeID2,
				Quantity:     3,
			},
		},
	}

	resp, err := service.CreateOrder(ctx, userID, req)
	if err != nil {
		t.Fatalf("unexpected error creating order: %v", err)
	}

	// Verify Order Response
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	expectedTotal := int64(2*500000 + 3*200000) // 1,600,000
	if resp.TotalAmount != expectedTotal {
		t.Errorf("expected total amount %d, got %d", expectedTotal, resp.TotalAmount)
	}
	if resp.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", resp.Status)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 order items in response, got %d", len(resp.Items))
	}

	// Verify Outbox Event was created properly
	if mockDB.capturedEventType != events.TopicOrderEvents {
		t.Errorf("expected event type %s, got %s", events.TopicOrderEvents, mockDB.capturedEventType)
	}
	if mockDB.capturedAggregateID != resp.ID.String() {
		t.Errorf("expected aggregate_id %s, got %s", resp.ID.String(), mockDB.capturedAggregateID)
	}

	// Verify Payload is valid JSON and contains order info
	var payloadData dto.OrderResponse
	if err := json.Unmarshal([]byte(mockDB.capturedOutboxPayload), &payloadData); err != nil {
		t.Fatalf("failed to unmarshal outbox payload: %v", err)
	}
	if payloadData.ID != resp.ID {
		t.Errorf("payload ID %s does not match order ID %s", payloadData.ID, resp.ID)
	}
	if payloadData.TotalAmount != expectedTotal {
		t.Errorf("payload TotalAmount %d does not match expected %d", payloadData.TotalAmount, expectedTotal)
	}
	if len(payloadData.Items) != 2 {
		t.Errorf("payload items length %d, expected 2", len(payloadData.Items))
	}
}

func TestCreateOrder_MergeDuplicateTicketTypeIDs(t *testing.T) {
	mockDB := &mockDBTX{}
	queries := db.New(mockDB)

	repo := &mockOrderRepo{
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			return fn(queries)
		},
	}

	ticketTypeID1 := uuid.New()
	ticketTypeID2 := uuid.New()

	var capturedDeductItems []*pbInventory.DeductItem
	mockInvClient := &mockInventoryClient{
		deductStockFn: func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
			capturedDeductItems = in.Items
			return &pbInventory.DeductStockResponse{Success: true}, nil
		},
	}

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{
						Id:       ticketTypeID1.String(),
						Name:     "VIP Ticket",
						Sku:      "VIP-001",
						Price:    100000,
						Currency: "VND",
					},
					{
						Id:       ticketTypeID2.String(),
						Name:     "Standard Ticket",
						Sku:      "STD-001",
						Price:    50000,
						Currency: "VND",
					},
				},
			}, nil
		},
	}

	service := NewOrderService(repo, nil, mockInvClient, eventClient)
	ctx := context.Background()

	// Client sends duplicate ticketTypeID1 (2 items with quantities 2 and 3)
	req := dto.CreateOrderRequest{
		Currency: "VND",
		Items: []dto.CreateOrderItemRequest{
			{TicketTypeID: ticketTypeID1, Quantity: 2},
			{TicketTypeID: ticketTypeID1, Quantity: 3},
			{TicketTypeID: ticketTypeID2, Quantity: 1},
		},
	}

	resp, err := service.CreateOrder(ctx, uuid.New(), req)
	if err != nil {
		t.Fatalf("unexpected error creating order: %v", err)
	}

	// Expected: ticketTypeID1 merged to 5 qty (5 * 100k = 500k), ticketTypeID2 is 1 qty (50k) => total 550,000
	expectedTotal := int64(5*100000 + 1*50000)
	if resp.TotalAmount != expectedTotal {
		t.Errorf("expected total amount %d, got %d", expectedTotal, resp.TotalAmount)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 merged items in response, got %d", len(resp.Items))
	}
	if resp.Items[0].Quantity != 5 {
		t.Errorf("expected merged item 0 quantity 5, got %d", resp.Items[0].Quantity)
	}

	// Verify inventory deduction got merged items
	if len(capturedDeductItems) != 2 {
		t.Fatalf("expected 2 items sent to inventory, got %d", len(capturedDeductItems))
	}
	if capturedDeductItems[0].Quantity != 5 {
		t.Errorf("expected item 0 quantity 5 sent to inventory, got %d", capturedDeductItems[0].Quantity)
	}
}

func TestCreateOrder_InventoryInsufficientStock(t *testing.T) {
	ticketTypeID := uuid.New()
	mockClient := &mockInventoryClient{
		deductStockFn: func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
			return nil, grpcErrors.InsufficientStock(ticketTypeID.String())
		},
	}

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{
						Id:       ticketTypeID.String(),
						Name:     "Test Ticket",
						Sku:      "TEST-01",
						Price:    1000,
						Currency: "VND",
					},
				},
			}, nil
		},
	}

	dbTxCalled := false
	repo := &mockOrderRepo{
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			dbTxCalled = true
			return nil
		},
	}

	service := NewOrderService(repo, nil, mockClient, eventClient)
	ctx := context.Background()

	_, err := service.CreateOrder(ctx, uuid.New(), dto.CreateOrderRequest{
		Currency: "VND",
		Items: []dto.CreateOrderItemRequest{
			{TicketTypeID: ticketTypeID, Quantity: 5},
		},
	})

	if !errors.Is(err, dto.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}

	// Verify DB transaction was never executed!
	if dbTxCalled {
		t.Fatalf("DB transaction was executed despite insufficient stock!")
	}
}

func TestCreateOrder_InventoryClientDeductItemsPassed(t *testing.T) {
	ticketTypeID1 := uuid.New()
	ticketTypeID2 := uuid.New()

	var capturedItems []*pbInventory.DeductItem
	mockClient := &mockInventoryClient{
		deductStockFn: func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
			capturedItems = in.Items
			return &pbInventory.DeductStockResponse{Success: true}, nil
		},
	}

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{
						Id:       ticketTypeID1.String(),
						Name:     "Ticket 1",
						Sku:      "T1",
						Price:    100,
						Currency: "VND",
					},
					{
						Id:       ticketTypeID2.String(),
						Name:     "Ticket 2",
						Sku:      "T2",
						Price:    200,
						Currency: "VND",
					},
				},
			}, nil
		},
	}

	mockDB := &mockDBTX{}
	queries := db.New(mockDB)
	repo := &mockOrderRepo{
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			return fn(queries)
		},
	}

	service := NewOrderService(repo, nil, mockClient, eventClient)
	ctx := context.Background()

	_, err := service.CreateOrder(ctx, uuid.New(), dto.CreateOrderRequest{
		Currency: "VND",
		Items: []dto.CreateOrderItemRequest{
			{TicketTypeID: ticketTypeID1, Quantity: 2},
			{TicketTypeID: ticketTypeID2, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedItems) != 2 {
		t.Fatalf("expected 2 items sent to inventory service, got %d", len(capturedItems))
	}
	if capturedItems[0].TicketTypeId != ticketTypeID1.String() || capturedItems[0].Quantity != 2 {
		t.Errorf("item 0 mismatch: %+v", capturedItems[0])
	}
	if capturedItems[1].TicketTypeId != ticketTypeID2.String() || capturedItems[1].Quantity != 3 {
		t.Errorf("item 1 mismatch: %+v", capturedItems[1])
	}
}

func TestCreateOrder_CompensatesStockOnTransactionError(t *testing.T) {
	ticketTypeID1 := uuid.New()
	ticketTypeID2 := uuid.New()

	var restoredItems []*pbInventory.RestoreItem
	restoreCalled := false
	mockClient := &mockInventoryClient{
		deductStockFn: func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
			return &pbInventory.DeductStockResponse{Success: true}, nil
		},
		restoreStockFn: func(ctx context.Context, in *pbInventory.RestoreStockRequest, opts ...grpc.CallOption) (*pbInventory.RestoreStockResponse, error) {
			restoreCalled = true
			restoredItems = in.Items
			return &pbInventory.RestoreStockResponse{Success: true}, nil
		},
	}

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{Id: ticketTypeID1.String(), Name: "T1", Sku: "SKU-1", Price: 10000, Currency: "VND"},
					{Id: ticketTypeID2.String(), Name: "T2", Sku: "SKU-2", Price: 20000, Currency: "VND"},
				},
			}, nil
		},
	}

	txErr := errors.New("db deadlock")
	repo := &mockOrderRepo{
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			return txErr
		},
	}

	service := NewOrderService(repo, nil, mockClient, eventClient)
	_, err := service.CreateOrder(context.Background(), uuid.New(), dto.CreateOrderRequest{
		Currency: "VND",
		Items: []dto.CreateOrderItemRequest{
			{TicketTypeID: ticketTypeID1, Quantity: 2},
			{TicketTypeID: ticketTypeID1, Quantity: 1},
			{TicketTypeID: ticketTypeID2, Quantity: 4},
		},
	})
	if !errors.Is(err, txErr) {
		t.Fatalf("expected txErr, got %v", err)
	}
	if !restoreCalled {
		t.Fatalf("expected RestoreStock to be called after DB transaction failure")
	}
	if len(restoredItems) != 2 {
		t.Fatalf("expected 2 merged restore items, got %d", len(restoredItems))
	}
	if restoredItems[0].TicketTypeId != ticketTypeID1.String() || restoredItems[0].Quantity != 3 {
		t.Errorf("unexpected restored item 0: %+v", restoredItems[0])
	}
	if restoredItems[1].TicketTypeId != ticketTypeID2.String() || restoredItems[1].Quantity != 4 {
		t.Errorf("unexpected restored item 1: %+v", restoredItems[1])
	}
}

func TestCreateOrder_CompensatesStockWithNonCancelledContext(t *testing.T) {
	ticketTypeID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())

	var restoreCtxErr error
	restoreCalled := false
	mockClient := &mockInventoryClient{
		restoreStockFn: func(rCtx context.Context, in *pbInventory.RestoreStockRequest, opts ...grpc.CallOption) (*pbInventory.RestoreStockResponse, error) {
			restoreCalled = true
			restoreCtxErr = rCtx.Err()
			return &pbInventory.RestoreStockResponse{Success: true}, nil
		},
	}

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{Id: ticketTypeID.String(), Name: "T1", Sku: "SKU-1", Price: 10000, Currency: "VND"},
				},
			}, nil
		},
	}

	repo := &mockOrderRepo{
		execTxFn: func(txCtx context.Context, fn func(q *db.Queries) error) error {
			cancel()
			return context.Canceled
		},
	}

	service := NewOrderService(repo, nil, mockClient, eventClient)
	_, err := service.CreateOrder(ctx, uuid.New(), dto.CreateOrderRequest{
		Items: []dto.CreateOrderItemRequest{{TicketTypeID: ticketTypeID, Quantity: 2}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !restoreCalled {
		t.Fatalf("expected RestoreStock to be called even when ctx was cancelled")
	}
	if restoreCtxErr != nil {
		t.Fatalf("expected non-cancelled context for RestoreStock, got %v", restoreCtxErr)
	}
}

func TestCreateOrder_RestoreStockErrorDoesNotMaskTxError(t *testing.T) {
	ticketTypeID := uuid.New()
	mockClient := &mockInventoryClient{
		restoreStockFn: func(ctx context.Context, in *pbInventory.RestoreStockRequest, opts ...grpc.CallOption) (*pbInventory.RestoreStockResponse, error) {
			return nil, errors.New("inventory gRPC unavailable")
		},
	}

	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{Id: ticketTypeID.String(), Name: "T1", Sku: "SKU-1", Price: 10000, Currency: "VND"},
				},
			}, nil
		},
	}

	txErr := errors.New("db insert failed")
	repo := &mockOrderRepo{
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			return txErr
		},
	}

	service := NewOrderService(repo, nil, mockClient, eventClient)
	_, err := service.CreateOrder(context.Background(), uuid.New(), dto.CreateOrderRequest{
		Items: []dto.CreateOrderItemRequest{{TicketTypeID: ticketTypeID, Quantity: 1}},
	})
	if !errors.Is(err, txErr) {
		t.Fatalf("expected original txErr, got %v", err)
	}
}

func TestCreateOrder_IdempotencyKey_ReplaysCachedResponse(t *testing.T) {
	mockDB := &mockDBTX{}
	queries := db.New(mockDB)
	txCalls := 0
	repo := &mockOrderRepo{
		queriesFn: func() *db.Queries { return queries },
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			txCalls++
			return fn(queries)
		},
	}

	ticketTypeID := uuid.New()
	eventCalls := 0
	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			eventCalls++
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{Id: ticketTypeID.String(), Name: "VIP", Sku: "VIP-1", Price: 150000, Currency: "VND"},
				},
			}, nil
		},
	}

	deductCalls := 0
	invClient := &mockInventoryClient{
		deductStockFn: func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
			deductCalls++
			return &pbInventory.DeductStockResponse{Success: true}, nil
		},
	}

	service := NewOrderService(repo, nil, invClient, eventClient)
	userID := uuid.New()
	req := dto.CreateOrderRequest{
		IdempotencyKey: "idem-key-001",
		Currency:       "VND",
		Items:          []dto.CreateOrderItemRequest{{TicketTypeID: ticketTypeID, Quantity: 2}},
	}

	firstResp, err := service.CreateOrder(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("first CreateOrder failed: %v", err)
	}

	secondResp, err := service.CreateOrder(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("second CreateOrder failed: %v", err)
	}

	if secondResp.ID != firstResp.ID || secondResp.OrderCode != firstResp.OrderCode {
		t.Fatalf("expected replayed order %s (%s), got %s (%s)", firstResp.ID, firstResp.OrderCode, secondResp.ID, secondResp.OrderCode)
	}
	if eventCalls != 1 || deductCalls != 1 || txCalls != 1 {
		t.Fatalf("expected downstream/tx called once, got event=%d deduct=%d tx=%d", eventCalls, deductCalls, txCalls)
	}

	// Different user with the same key must create a separate order (scoped by user_id)
	otherUserResp, err := service.CreateOrder(context.Background(), uuid.New(), req)
	if err != nil {
		t.Fatalf("other user CreateOrder failed: %v", err)
	}
	if otherUserResp.ID == firstResp.ID {
		t.Fatalf("expected different user to receive distinct order ID")
	}
}

func TestCreateOrder_IdempotencyKey_InProgressReturnsConflict(t *testing.T) {
	userID := uuid.New()
	key := "idem-in-progress"
	mockDB := &mockDBTX{
		idempotencyKeys: map[string]db.OrderIdempotencyKey{
			userID.String() + ":" + key: {
				UserID:    userID,
				Key:       key,
				OrderID:   uuid.New(), // pre-generated, order not committed in DB yet
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
		},
	}
	queries := db.New(mockDB)
	repo := &mockOrderRepo{
		queriesFn: func() *db.Queries { return queries },
	}

	deductCalled := false
	invClient := &mockInventoryClient{
		deductStockFn: func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
			deductCalled = true
			return &pbInventory.DeductStockResponse{Success: true}, nil
		},
	}

	service := NewOrderService(repo, nil, invClient, &mockEventClient{})
	_, err := service.CreateOrder(context.Background(), userID, dto.CreateOrderRequest{
		IdempotencyKey: key,
		Items:          []dto.CreateOrderItemRequest{{TicketTypeID: uuid.New(), Quantity: 1}},
	})
	if !errors.Is(err, dto.ErrOrderInProgress) {
		t.Fatalf("expected ErrOrderInProgress, got %v", err)
	}
	if deductCalled {
		t.Fatalf("DeductStock must not be called when request is already in progress")
	}
}

func TestCreateOrder_IdempotencyKey_ReleasedOnFailureAllowsRetry(t *testing.T) {
	mockDB := &mockDBTX{}
	queries := db.New(mockDB)
	txAttempt := 0
	repo := &mockOrderRepo{
		queriesFn: func() *db.Queries { return queries },
		execTxFn: func(ctx context.Context, fn func(q *db.Queries) error) error {
			txAttempt++
			if txAttempt == 1 {
				return errors.New("temporary db error")
			}
			return fn(queries)
		},
	}

	ticketTypeID := uuid.New()
	eventClient := &mockEventClient{
		getTicketTypesFn: func(ctx context.Context, in *pbEvent.GetTicketTypesRequest, opts ...grpc.CallOption) (*pbEvent.GetTicketTypesResponse, error) {
			return &pbEvent.GetTicketTypesResponse{
				TicketTypes: []*pbEvent.TicketTypeDetail{
					{Id: ticketTypeID.String(), Name: "T1", Sku: "SKU-1", Price: 10000, Currency: "VND"},
				},
			}, nil
		},
	}

	service := NewOrderService(repo, nil, &mockInventoryClient{}, eventClient)
	userID := uuid.New()
	req := dto.CreateOrderRequest{
		IdempotencyKey: "idem-retry-after-fail",
		Items:          []dto.CreateOrderItemRequest{{TicketTypeID: ticketTypeID, Quantity: 1}},
	}

	if _, err := service.CreateOrder(context.Background(), userID, req); err == nil {
		t.Fatalf("expected first attempt to fail")
	}

	// Retry with same key must succeed because failed attempt released the key
	resp, err := service.CreateOrder(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("expected retry with same key to succeed, got %v", err)
	}
	if resp == nil || resp.ID == uuid.Nil {
		t.Fatalf("expected valid order response on retry")
	}
}

func TestCreateOrder_IdempotencyKey_TooLong(t *testing.T) {
	service := NewOrderService(&mockOrderRepo{}, nil, &mockInventoryClient{}, &mockEventClient{})
	_, err := service.CreateOrder(context.Background(), uuid.New(), dto.CreateOrderRequest{
		IdempotencyKey: strings.Repeat("a", MaxIdempotencyKeyLen+1),
		Items:          []dto.CreateOrderItemRequest{{TicketTypeID: uuid.New(), Quantity: 1}},
	})
	if !errors.Is(err, dto.ErrInvalidIdempotencyKey) {
		t.Fatalf("expected ErrInvalidIdempotencyKey, got %v", err)
	}
}
