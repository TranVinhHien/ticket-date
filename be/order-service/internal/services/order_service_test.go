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
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type mockInventoryClient struct {
	deductStockFn func(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error)
}

func (m *mockInventoryClient) DeductStock(ctx context.Context, in *pbInventory.DeductStockRequest, opts ...grpc.CallOption) (*pbInventory.DeductStockResponse, error) {
	if m.deductStockFn != nil {
		return m.deductStockFn(ctx, in, opts...)
	}
	return &pbInventory.DeductStockResponse{Success: true}, nil
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
	execTxFn func(ctx context.Context, fn func(q *db.Queries) error) error
}

func (m *mockOrderRepo) Queries() *db.Queries {
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

func (m *mockDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (m *mockDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
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
				*dest[9].(*time.Time) = time.Now()
				*dest[10].(**uuid.UUID) = createdBy
				*dest[11].(*time.Time) = time.Now()
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
	if mockDB.capturedEventType != EventOrderCreated {
		t.Errorf("expected event type %s, got %s", EventOrderCreated, mockDB.capturedEventType)
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
