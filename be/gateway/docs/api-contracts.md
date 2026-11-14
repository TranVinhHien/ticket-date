# Gateway - API Routing & Authorization Contracts

---

## 1. Gateway Native Endpoints

| Method | Endpoint | Auth Required | Description | Response Status |
|---|---|---|---|---|
| `GET` | `/health` | No | Gateway liveness check | `200 OK` (`{"service":"gateway","status":"OK",...}`) |

---

## 2. Downstream Proxy Routes (From `gateway.example.yaml`)

All non-native requests are mapped to upstream targets using prefix matching:

| Route Prefix | Upstream Target | Auth Required | Target Service |
|---|---|---|---|
| `/api/v1/auth/**` | `http://localhost:8080` | `false` | `user` |
| `/api/v1/users/**` | `http://localhost:8080` | `true` | `user` |
| `/api/v1/upload/**` | `http://localhost:8080` | `true` | `user` |
---

## 3. RBAC Policy Matrix (From `permissions.json`)

The Gateway evaluates role permissions in `middlewares.AuthMiddleware` using `permissions.Evaluator`:

### Role: `admin`
- Full wildcard access: `GET: ["*"]`, `POST: ["*"]`, `PUT: ["*"]`, `PATCH: ["*"]`, `DELETE: ["*"]`.

### Role: `user`
- `GET`: `/api/v1/users/*`, `/api/v1/catalog/*`, `/api/v1/library/*`, `/api/v1/audio/*`
- `POST`: `/api/v1/library/*`, `/api/v1/listening-events/*`, `/api/v1/upload`
- `PUT`: `/api/v1/users/*`, `/api/v1/library/*`
- `DELETE`: `/api/v1/library/*`

---

## 4. Error Responses
- **401 Unauthorized**: Missing or malformed `Authorization: Bearer <token>`, invalid signature, or expired token.
- **403 Forbidden**: Token valid, but user role is not permitted for the requested HTTP method and path in `permissions.json`.
- **404 Not Found**: Path does not match any registered route in `gateway.*.yaml`.
- **502 Bad Gateway**: Downstream target is unreachable or timed out.
