# Event Service

Event Service is responsible for managing event organizers, venue locations, event metadata & lifecycles, and event ticket tiers for the Ticket-Date ticketing platform.

## Features
- **Organizers**: CRUD operations, status management (`ACTIVE`, `INACTIVE`), soft deletes.
- **Venues**: CRUD operations, geo-coordinates, capacity validation, organizer links.
- **Events**: CRUD operations, slug uniqueness / auto-generation, date interval validation, status lifecycle (`DRAFT`, `PUBLISHED`, `CANCELLED`, `COMPLETED`), fetch by ID & slug.
- **Ticket Types**: Pricing in VND/currency, sales period bounds, max order constraints, custom JSONB attributes.

## Quick Start

### 1. Configuration
Copy `app.example.yaml` to `app.yaml`:
```bash
cp app.example.yaml app.yaml
```

Environment variables:
- `PORT` or `EVENT_SERVICE_PORT`: Server port (default `8081`)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: Database credentials
- `JWT_SECRET`: Token secret key

### 2. Run Database Migrations & Server
```bash
go run ./cmd/server
```

### 3. Run Tests
- **Unit Tests**:
  ```bash
  go test -v ./internal/services/...
  ```
- **Integration Tests**:
  ```bash
  go test -v -tags integration ./internal/integration/...
  ```
