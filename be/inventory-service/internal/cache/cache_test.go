package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/cache"
)

type sampleData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

func TestMemoryCache_BasicOperations(t *testing.T) {
	c := cache.NewMemoryCache(10*time.Minute, 1*time.Minute)
	ctx := context.Background()

	// 1. Get non-existent key
	var out sampleData
	err := c.Get(ctx, "non_existent", &out)
	if err != cache.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}

	// 2. Set key and Get
	data := sampleData{ID: "123", Name: "VIP Ticket", Qty: 50}
	err = c.Set(ctx, "ticket:123", data, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error on Set: %v", err)
	}

	err = c.Get(ctx, "ticket:123", &out)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if out.ID != data.ID || out.Qty != data.Qty {
		t.Fatalf("expected %+v, got %+v", data, out)
	}

	// 3. Exists
	exists, err := c.Exists(ctx, "ticket:123")
	if err != nil || !exists {
		t.Fatalf("expected key to exist, err: %v", err)
	}

	// 4. Delete
	err = c.Delete(ctx, "ticket:123")
	if err != nil {
		t.Fatalf("unexpected error on Delete: %v", err)
	}

	err = c.Get(ctx, "ticket:123", &out)
	if err != cache.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss after Delete, got %v", err)
	}
}

func TestMemoryCache_TTL_Expiration(t *testing.T) {
	c := cache.NewMemoryCache(10*time.Minute, 1*time.Minute)
	ctx := context.Background()

	data := sampleData{ID: "exp", Name: "Expire Soon", Qty: 10}
	err := c.Set(ctx, "ticket:exp", data, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected Set error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	var out sampleData
	err = c.Get(ctx, "ticket:exp", &out)
	if err != cache.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss due to TTL expiration, got %v", err)
	}
}

func TestMemoryCache_Remember(t *testing.T) {
	c := cache.NewMemoryCache(10*time.Minute, 1*time.Minute)
	ctx := context.Background()

	called := 0
	fetcher := func() (any, error) {
		called++
		return sampleData{ID: "rem", Name: "Remembered", Qty: 99}, nil
	}

	var res1 sampleData
	err := c.Remember(ctx, "ticket:rem", &res1, 5*time.Minute, fetcher)
	if err != nil {
		t.Fatalf("unexpected Remember error: %v", err)
	}
	if res1.Qty != 99 || called != 1 {
		t.Fatalf("expected fetcher called once with Qty 99, got called=%d, Qty=%d", called, res1.Qty)
	}

	// Second call should hit cache and NOT call fetcher
	var res2 sampleData
	err = c.Remember(ctx, "ticket:rem", &res2, 5*time.Minute, fetcher)
	if err != nil {
		t.Fatalf("unexpected second Remember error: %v", err)
	}
	if res2.Qty != 99 || called != 1 {
		t.Fatalf("expected fetcher NOT called on second call, got called=%d", called)
	}
}

func TestMemoryCache_IntegerCounterOperations(t *testing.T) {
	c := cache.NewMemoryCache(10*time.Minute, 1*time.Minute)
	ctx := context.Background()

	// 1. Get non-existent int
	val, err := c.GetInt(ctx, "inv:available:1")
	if err != cache.ErrCacheMiss || val != 0 {
		t.Fatalf("expected ErrCacheMiss, got val=%d, err=%v", val, err)
	}

	// 2. SetInt & GetInt
	err = c.SetInt(ctx, "inv:available:1", 50, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected SetInt error: %v", err)
	}

	val, err = c.GetInt(ctx, "inv:available:1")
	if err != nil || val != 50 {
		t.Fatalf("expected val=50, got val=%d, err=%v", val, err)
	}

	// 3. IncrBy (+)
	newVal, err := c.IncrBy(ctx, "inv:available:1", 10)
	if err != nil || newVal != 60 {
		t.Fatalf("expected 60 after +10, got %d, err=%v", newVal, err)
	}

	// 4. DecrBy (-)
	newVal, err = c.DecrBy(ctx, "inv:available:1", 25)
	if err != nil || newVal != 35 {
		t.Fatalf("expected 35 after -25, got %d, err=%v", newVal, err)
	}

	// 5. Verify final value
	val, err = c.GetInt(ctx, "inv:available:1")
	if err != nil || val != 35 {
		t.Fatalf("expected final val=35, got %d, err=%v", val, err)
	}
}
