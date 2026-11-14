# Gateway - Context & Documentation Index

This directory contains modular documentation for **`gateway`**. Select the appropriate document based on your task:

| Document | When to Read / Purpose |
|---|---|
| [architecture.md](file:///Users/hienlazada/ticket-date/be/gateway/docs/architecture.md) | Internal structure, Gin engine lifecycle, ReverseProxyHandler mechanics, and middleware execution order. |
| [api-contracts.md](file:///Users/hienlazada/ticket-date/be/gateway/docs/api-contracts.md) | Downstream route mappings, public vs protected routes, and RBAC matrix from `permissions.json`. |
| [../AGENTS.md](file:///Users/hienlazada/ticket-date/be/gateway/AGENTS.md) | Agent operational rules, constraints (statelessness), testing commands, and implementation DOs/DON'Ts. |

---

## Service Status
- **Implementation Status**: Fully implemented and functional.
- **Upstream Targets**:
  - `user`: Active on `http://localhost:8080`.
