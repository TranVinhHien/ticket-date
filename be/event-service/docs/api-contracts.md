# API Contracts - Event Service

Base URL: `/api/v1`

## Standard Envelope
```json
{
  "code": "200",
  "message": "ok",
  "meta": {
    "current_page": 1,
    "page_size": 10,
    "total_items": 1,
    "total_pages": 1,
    "has_next": false,
    "has_previous": false
  },
  "data": {}
}
```

---

## 1. Organizers API
- `POST /api/v1/organizers`: Create organizer
- `GET /api/v1/organizers`: List organizers (paginated)
- `GET /api/v1/organizers/:id`: Get organizer by ID
- `PUT /api/v1/organizers/:id`: Update organizer
- `DELETE /api/v1/organizers/:id`: Soft delete organizer

---

## 2. Venues API
- `POST /api/v1/venues`: Create venue (with capacity, coordinates)
- `GET /api/v1/venues`: List venues (paginated, filtered by organizer/city)
- `GET /api/v1/venues/:id`: Get venue by ID
- `PUT /api/v1/venues/:id`: Update venue
- `DELETE /api/v1/venues/:id`: Soft delete venue

---

## 3. Events & Ticket Types API
Ticket types are managed directly within the Event aggregate.

### Create Event with Ticket Types
- `POST /api/v1/events`
  - Body:
    ```json
    {
      "organizer_id": "uuid (required)",
      "venue_id": "uuid (required)",
      "name": "string (required)",
      "slug": "string (optional, auto-generated from name if omitted)",
      "description": "string",
      "poster_url": "string",
      "start_at": "RFC3339 timestamp (required)",
      "end_at": "RFC3339 timestamp (required, must be after start_at)",
      "status": "DRAFT | PUBLISHED | CANCELLED | COMPLETED",
      "ticket_types": [
        {
          "sku": "string (required, unique)",
          "name": "string (required)",
          "description": "string",
          "price": 2500000,
          "currency": "VND",
          "max_per_order": 4,
          "sale_start_at": "RFC3339 timestamp (required)",
          "sale_end_at": "RFC3339 timestamp (required, must be after sale_start_at)",
          "attributes": { "zone": "VIP" },
          "status": "ACTIVE | DRAFT | PAUSED | ENDED"
        }
      ]
    }
    ```
  - Response: 201 Created (includes event and created ticket_types)

### Query Events
- `GET /api/v1/events`: List events with pagination, query, status, organizer_id, venue_id (includes ticket_types).
- `GET /api/v1/events/:id`: Get event by ID (preloads organizer, venue, and active ticket_types).
- `GET /api/v1/events/slug/:slug`: Get event by slug (preloads organizer, venue, and active ticket_types).

### Update Event & Ticket Types
- `PUT /api/v1/events/:id`
  - Body:
    ```json
    {
      "name": "string",
      "description": "string",
      "ticket_types": [
        {
          "id": "uuid (optional: if provided updates existing ticket; if omitted creates new)",
          "price": 2800000,
          "description": "Updated description",
          "is_deleted": false
        },
        {
          "id": "uuid",
          "is_deleted": true
        }
      ]
    }
    ```
  - Response: 200 OK (includes updated event and active ticket_types)

### Delete Event
- `DELETE /api/v1/events/:id`: Soft-deletes the event and cascades soft-delete to all associated ticket types.
