package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mockDBTX struct {
	queryFn func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

func (m *mockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return &mockRows{}, nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}

type mockRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *mockRows) Close()                                       {}
func (r *mockRows) Err() error                                   { return r.err }
func (r *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *mockRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = row[i].(uuid.UUID)
		case *string:
			*d = row[i].(string)
		case *int64:
			*d = row[i].(int64)
		case **time.Time:
			if row[i] != nil {
				*d = row[i].(*time.Time)
			}
		case **uuid.UUID:
			if row[i] != nil {
				*d = row[i].(*uuid.UUID)
			}
		case *time.Time:
			*d = row[i].(time.Time)
		}
	}
	return nil
}
func (r *mockRows) Values() ([]any, error) { return nil, nil }
func (r *mockRows) RawValues() [][]byte    { return nil }
func (r *mockRows) Conn() *pgx.Conn        { return nil }
func (r *mockRows) TypeMap() *pgtype.Map   { return nil }

type mockOrderRepo struct {
	queries *db.Queries
}

func (m *mockOrderRepo) Queries() *db.Queries { return m.queries }
func (m *mockOrderRepo) GetPool() *pgxpool.Pool { return nil }
func (m *mockOrderRepo) ExecTx(ctx context.Context, fn func(q *db.Queries) error) error {
	return nil
}

func TestPendingOrdersJob_Run_Success(t *testing.T) {
	now := time.Now()
	orderID := uuid.New()
	userID := uuid.New()

	var capturedStatus string
	var capturedLimit int32
	var capturedOffset int32

	mockDB := &mockDBTX{
		queryFn: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedStatus = args[0].(string)
			capturedLimit = args[1].(int32)
			capturedOffset = args[2].(int32)
			return &mockRows{
				rows: [][]any{
					{
						orderID,
						userID,
						"ORD-20261126-TEST01",
						int64(150000),
						"VND",
						OrderStatusPending,
						nil,
						nil,
						nil,
						now,
						nil,
						now,
						nil,
						nil,
					},
				},
			}, nil
		},
	}

	repo := &mockOrderRepo{queries: db.New(mockDB)}
	job := NewPendingOrdersJob(repo, 30*time.Second, 50)

	if job.Name() != "pending-orders" {
		t.Fatalf("unexpected job name: %s", job.Name())
	}
	if job.Interval() != 30*time.Second {
		t.Fatalf("unexpected interval: %s", job.Interval())
	}

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedStatus != OrderStatusPending {
		t.Fatalf("expected status %q, got %q", OrderStatusPending, capturedStatus)
	}
	if capturedLimit != 50 || capturedOffset != 0 {
		t.Fatalf("expected limit=50 offset=0, got limit=%d offset=%d", capturedLimit, capturedOffset)
	}
}

func TestPendingOrdersJob_Run_DBError(t *testing.T) {
	mockDB := &mockDBTX{
		queryFn: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, errors.New("db connection error")
		},
	}

	repo := &mockOrderRepo{queries: db.New(mockDB)}
	job := NewPendingOrdersJob(repo, 0, 0)

	if err := job.Run(context.Background()); err == nil {
		t.Fatal("expected error when DB query fails")
	}
}
