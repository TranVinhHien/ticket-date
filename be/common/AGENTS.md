---
trigger: model_decision
description: "Guidelines and rules for the shared foundational library module common."
---

# Agent Guidelines - Common Library

## 1. Module Responsibility

### What it does
- **Single Source of Shared Code**: Provides cross-service data types, generic response envelopes, pagination structures, canonical header constants, and reusable HTTP clients.
- **Pure Library Package**: Exclusively exports reusable Go packages. Contains zero business logic, zero runnable main daemons, and zero database connections.

### What it does NOT do
- No executable main entry points (`cmd/` does not exist here).
- No direct database schema definitions or migrations.
- No service-specific business rules.

---

## 2. Package Catalog

- **`constants`**: Centralized HTTP header names (`HeaderUserID`, `HeaderRequestID`, `HeaderUserRoles`, etc.) and context keys.
- **`response`**: Standardized JSON envelopes (`Response[T]`, `AnyResponse`) and helper response functions.
- **`pagination`**: Unified pagination metadata and request parameter parsers.
- **`httpclient`**: Resilient internal HTTP client with timeout and header injection capabilities.
- **`middleware`**: Shared Gin middleware functions.
- **`grpc`**: Reusable gRPC framework containing:
  - `grpc/context`: Seamless conversion between Gin/Go Context and gRPC Metadata.
  - `grpc/interceptor`: Standardized Unary and Stream interceptors (Recovery, Trace, Auth, Zerolog logging).
  - `grpc/client`: Managed connection factory (`NewClientConn`) with KeepAlive, timeout, and interceptors.
  - `grpc/errors`: Standard gRPC error builders and bidirectional HTTP status mappers.
  - `grpc/helper`: Utilities for Protobuf `structpb` and `timestamppb`.
  - `grpc/proto` & `grpc/pb`: Centralized protobuf definitions and generated stubs.
- **`kafka`**: Shared production-grade Kafka infrastructure using `twmb/franz-go`:
  - `kafka.Producer` / `franzProducer`: High-throughput, idempotent async/sync producer with `EventEnvelope` wrapping, LZ4 batch compression, and SASL support.
  - `kafka.Consumer` / `franzConsumer`: Consumer group with per-partition ordered concurrency workers, in-memory exponential backoff retry, and optional Dead Letter Queue (`.dlq`).
  - `kafka.EventEnvelope`: Canonical cross-service event wrapper (`event_id`, `event_type`, `version`, `source`, `occurred_at`, `trace_id`, `data`).
  - `kafka/events`: Strongly-typed payloads for inter-service domain events (e.g. `OrderCreatedPayload`, `OrderCreatedItem`).

---

## 3. Implementation Rules

1. **Zero External Service Dependencies**: `common` must never import packages from any microservice (`gateway`, `user`, etc.).
2. **Backward Compatibility**: Any change to struct fields in `response` or `constants` affects all services simultaneously. Keep changes strictly additive.
3. **Run Tests**: Always run `go test ./...` in `common` before committing modifications.
