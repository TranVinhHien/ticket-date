---
trigger: model_decision
description: "Guidelines, domain boundaries, data models, and implementation standards for event-service."
---

# Agent Guidelines - Event Service

## 1. Service Responsibility

### What it does (Core Responsibilities)
- **Organizers**: Manages organizations, tax info, status, contact details, and ownership.
- **Venues**: Manages event locations, geo-coordinates (latitude/longitude), capacity limits, and organizer relationships.
- **Events (Aggregate Root)**: Manages events and nested ticket types directly within event lifecycle operations (`POST /api/v1/events`, `PUT /api/v1/events/:id`).
- **Ticket Types**: Defined within an event (pricing, currency, quotas, max per order, sale periods, JSON attributes). Provides gRPC service (`GetTicketTypes` on port 50051) for batch ticket detail queries.

### What it does NOT do (Out of Scope)
- **No Order/Payment Processing**: Handled by `order-service` and `payment-service`.
- **No Ingress Routing**: Does not handle global SSL termination or rate limiting (handled by `gateway`).

---

## 2. Testing & Verification

### Unit Tests
```bash
go test -v ./internal/services/...
```

### Integration Tests
```bash
go test -v -tags integration ./internal/integration/...
```
