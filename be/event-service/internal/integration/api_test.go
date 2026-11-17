//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func request(t *testing.T, method, path string, body any, userID *string) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if userID != nil {
		req.Header.Set("X-User-ID", *userID)
	}
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func TestEventService_EndToEndLifecycle(t *testing.T) {
	adminID := uuid.New().String()

	// 1. Health check
	if w := request(t, http.MethodGet, "/health", nil, nil); w.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", w.Code)
	}

	// 2. Create Organizer
	orgName := fmt.Sprintf("LiveNation Vietnam %d", time.Now().UnixNano())
	w := request(t, http.MethodPost, "/api/v1/organizers", map[string]any{
		"organization_name": orgName,
		"description":       "Top event organizer",
		"address":           "123 Nguyen Hue",
		"status":            "ACTIVE",
	}, &adminID)
	if w.Code != http.StatusCreated {
		t.Fatalf("create organizer failed: %d %s", w.Code, w.Body.String())
	}
	var orgRes struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &orgRes)
	orgID := orgRes.Data.ID

	// 3. Create Venue
	venueName := fmt.Sprintf("Saigon Exhibition Center %d", time.Now().UnixNano())
	w = request(t, http.MethodPost, "/api/v1/venues", map[string]any{
		"organizer_id": orgID,
		"name":         venueName,
		"address":      "799 Nguyen Van Linh, District 7",
		"city":         "Ho Chi Minh City",
		"country":      "Vietnam",
		"capacity":     15000,
		"status":       "ACTIVE",
	}, &adminID)
	if w.Code != http.StatusCreated {
		t.Fatalf("create venue failed: %d %s", w.Code, w.Body.String())
	}
	var venueRes struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &venueRes)
	venueID := venueRes.Data.ID

	// 4. Create Event with inline Ticket Types
	eventName := fmt.Sprintf("Spring Festival %d", time.Now().UnixNano())
	eventSlug := fmt.Sprintf("spring-festival-%d", time.Now().UnixNano())
	startAt := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	endAt := time.Now().Add(52 * time.Hour).UTC().Format(time.RFC3339)
	sku1 := fmt.Sprintf("VIP-%d", time.Now().UnixNano())
	sku2 := fmt.Sprintf("GA-%d", time.Now().UnixNano())
	saleStartAt := time.Now().UTC().Format(time.RFC3339)
	saleEndAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	w = request(t, http.MethodPost, "/api/v1/events", map[string]any{
		"organizer_id": orgID,
		"venue_id":     venueID,
		"name":         eventName,
		"slug":         eventSlug,
		"description":  "Massive spring concert",
		"start_at":     startAt,
		"end_at":       endAt,
		"status":       "DRAFT",
		"ticket_types": []map[string]any{
			{
				"sku":           sku1,
				"name":          "VIP Golden Circle",
				"price":         2000000,
				"currency":      "VND",
				"max_per_order": 4,
				"sale_start_at": saleStartAt,
				"sale_end_at":   saleEndAt,
				"status":        "ACTIVE",
			},
			{
				"sku":           sku2,
				"name":          "General Admission",
				"price":         600000,
				"currency":      "VND",
				"max_per_order": 6,
				"sale_start_at": saleStartAt,
				"sale_end_at":   saleEndAt,
				"status":        "ACTIVE",
			},
		},
	}, &adminID)
	if w.Code != http.StatusCreated {
		t.Fatalf("create event failed: %d %s", w.Code, w.Body.String())
	}
	var eventRes struct {
		Data struct {
			ID          string `json:"id"`
			Slug        string `json:"slug"`
			TicketTypes []struct {
				ID    string `json:"id"`
				SKU   string `json:"sku"`
				Price int64  `json:"price"`
			} `json:"ticket_types"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &eventRes)
	eventID := eventRes.Data.ID
	if len(eventRes.Data.TicketTypes) != 2 {
		t.Fatalf("expected 2 ticket types returned, got %d", len(eventRes.Data.TicketTypes))
	}

	// 5. Get Event by ID
	w = request(t, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", eventID), nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get event by ID failed: %d %s", w.Code, w.Body.String())
	}

	// 6. Get Event by Slug
	w = request(t, http.MethodGet, fmt.Sprintf("/api/v1/events/slug/%s", eventSlug), nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get event by Slug failed: %d %s", w.Code, w.Body.String())
	}

	// 7. Update Event & Ticket Types
	vipTTID := eventRes.Data.TicketTypes[0].ID
	newPrice := int64(2200000)
	w = request(t, http.MethodPut, fmt.Sprintf("/api/v1/events/%s", eventID), map[string]any{
		"name": eventName + " [UPDATED]",
		"ticket_types": []map[string]any{
			{
				"id":    vipTTID,
				"price": newPrice,
			},
		},
	}, &adminID)
	if w.Code != http.StatusOK {
		t.Fatalf("update event failed: %d %s", w.Code, w.Body.String())
	}

	// 8. Delete Event
	w = request(t, http.MethodDelete, fmt.Sprintf("/api/v1/events/%s", eventID), nil, &adminID)
	if w.Code != http.StatusOK {
		t.Fatalf("delete event failed: %d %s", w.Code, w.Body.String())
	}

	// 9. Verify 404
	if w = request(t, http.MethodGet, fmt.Sprintf("/api/v1/events/%s", eventID), nil, nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted event, got %d", w.Code)
	}
}
