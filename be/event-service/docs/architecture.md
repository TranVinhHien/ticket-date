# Internal Architecture - Event Service

> **Update Notice**: Update this file when changing internal structures, event domain logic, database models, or Event Service APIs.

---

## 1. Directory Structure
```
event-service/
├── cmd/
│   └── server/
│       └── main.go       # HTTP server entry point and Dependency Injection (DI)
├── internal/
│   ├── config/           # Configuration loader (Viper, env vars)
│   ├── database/         # Database connection setup with GORM & Goose migrations
│   │   └── migrations/   # Embedded SQL migrations (001_init.sql)
│   ├── handlers/         # HTTP request controllers (organizer, venue, event)
│   │   └── dto/          # Request & response Data Transfer Objects
│   ├── integration/      # End-to-end integration tests (setup_test.go, api_test.go)
│   ├── middleware/       # Auth context extraction (Gateway headers & direct Bearer token)
│   ├── models/           # GORM database entities (Organizer, Venue, Event, TicketType)
│   ├── repository/       # Data access layer (GORM queries & transactions)
│   ├── router/           # Gin router configuration & route mounting
│   └── services/         # Domain business logic & validation
├── app.example.yaml      # Sample configuration
└── docs/
    ├── architecture.md   # This document
    ├── api-contracts.md  # Detailed API contracts
    └── data-model.md     # Relational database models and diagram
```

---

## 2. Core Business Flows

### A. Organizer & Venue Management
1. Organizers register company identity (`POST /api/v1/organizers`).
2. Venues are created linked to an organizer with physical capacity and geo-coordinates (`POST /api/v1/venues`).
3. Supports soft deletion ensuring historic audit preservation (`deleted_at`, `deleted_by`).

### B. Event Aggregate & Ticket Types
1. `Event` acts as the Aggregate Root. Ticket types are managed directly within the event lifecycle:
   - Creation (`POST /api/v1/events`): Event and nested `ticket_types` are persisted in a single transaction.
   - Retrieval (`GET /api/v1/events/:id` and `/slug/:slug`): Returns event with all active ticket types.
   - Update (`PUT /api/v1/events/:id`): Updates event properties and synchronizes ticket types (updating existing, adding new, soft-deleting removed).
   - Deletion (`DELETE /api/v1/events/:id`): Cascades soft-delete to the event and all associated ticket types.
