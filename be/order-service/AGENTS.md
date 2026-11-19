---
trigger: model_decision
description: "Guidelines, domain boundaries, data models, and implementation standards for order-service."
---

# Agent Guidelines - Order Service

## 1. Service Responsibility

### What it does (Core Responsibilities)
- **Orders**: Manages order creation, order tracking, order items, and lifecycle statuses (`PENDING`, `COMPLETED`, `CANCELLED`).
- **Create Order Flow**:
  1. Accepts client request with array of items (`ticket_type_id`, `quantity`).
  2. Aggregates duplicate `ticket_type_id` quantities.
  3. Queries `event-service` via gRPC (`GetTicketTypes` on port 50051) using batch query to fetch prices, names, and SKUs.
  4. Calls `inventory-service` via gRPC (`DeductStock` on port 50052) for atomic stock deduction.
  5. Atomically persists order, order items, and `outbox_event` inside a single DB transaction.
- **Outbox Pattern**: Emits `OrderCreated` events via `outbox_event` table for asynchronous messaging.

### What it does NOT do (Out of Scope)
- **No Stock Management**: Stock state is managed exclusively by `inventory-service`.
- **No Direct Event/Ticket Data Mutation**: Event details and ticket catalog are managed by `event-service`.
- **No Payment Processing**: Handled by `payment-service`.

---

## 2. Testing & Verification

### Unit Tests
```bash
go test -v ./internal/handlers ./internal/services
```
