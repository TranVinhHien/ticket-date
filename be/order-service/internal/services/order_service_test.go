package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	service := NewOrderService(&mockOrderRepo{}, nil)
	ctx := context.Background()
	validUserID := uuid.New()
	validTicketTypeID := uuid.New()

	tests := []struct {
		name        string
		userID      uuid.UUID
		req         dto.CreateOrderRequest
		expectedErr error
	}{
		{
			name:        "invalid user id (nil)",
			userID:      uuid.Nil,
			req:         dto.CreateOrderRequest{Items: []dto.CreateOrderItemRequest{{TicketTypeID: validTicketTypeID, Quantity: 1, UnitPrice: 1000}}},
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
					{TicketTypeID: validTicketTypeID, Quantity: 0, UnitPrice: 1000},
				},
			},
			expectedErr: dto.ErrInvalidQuantity,
		},
		{
			name:   "invalid unit price (< 0)",
			userID: validUserID,
			req: dto.CreateOrderRequest{
				Items: []dto.CreateOrderItemRequest{
					{TicketTypeID: validTicketTypeID, Quantity: 2, UnitPrice: -100},
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

	service := NewOrderService(repo, nil)
	ctx := context.Background()
	userID := uuid.New()
	ticketTypeID1 := uuid.New()
	ticketTypeID2 := uuid.New()

	req := dto.CreateOrderRequest{
		Currency: "VND",
		Items: []dto.CreateOrderItemRequest{
			{
				TicketTypeID:   ticketTypeID1,
				TicketTypeName: "VIP Ticket",
				Sku:            "VIP-001",
				Quantity:       2,
				UnitPrice:      500000,
			},
			{
				TicketTypeID:   ticketTypeID2,
				TicketTypeName: "Standard Ticket",
				Sku:            "STD-001",
				Quantity:       3,
				UnitPrice:      200000,
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
