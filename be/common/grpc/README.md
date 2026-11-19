# Common gRPC Package (`be/common/grpc`)

Shared gRPC utilities, interceptors, context propagation, error mappers, and connection managers for the `ticket-date` microservices ecosystem.

---

## 1. Directory Structure

```text
be/common/grpc/
├── client/         # Client connection manager (NewClientConn) with keepalive, retries & interceptors
├── context/        # Context <-> gRPC Metadata bridge (RequestID, UserID, UserRoles)
├── errors/         # Standard gRPC error constructors and HTTP status mappers
├── helper/         # Utilities for structpb.Struct and timestamppb.Timestamp
├── interceptor/    # Standard Unary & Stream Interceptors (Recovery, Trace, Auth, Logging)
├── pb/             # Generated Go code from Protobuf (.pb.go, _grpc.pb.go)
├── proto/          # Protocol Buffer schema definitions (*.proto)
├── Makefile        # Commands for generating protobufs and running tests
└── README.md       # Package documentation and usage examples
```

---

## 2. Server Implementation Example

To set up a gRPC server in any microservice (e.g., `inventory-service`):

```go
package main

import (
    "net"
    "github.com/rs/zerolog"
    "google.golang.org/grpc"
    "github.com/TranVinhHien/ticket-date/be/common/grpc/interceptor"
    pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
)

func main() {
    logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

    // 1. Initialize gRPC server with standard interceptors:
    // - Recovery: catches panics and returns codes.Internal
    // - Trace: ensures every request has a Request ID (X-Request-ID)
    // - Auth: binds X-User-ID, X-User-Email, X-User-Roles into context
    // - Logging: structured latency & status logging via Zerolog
    server := grpc.NewServer(
        grpc.ChainUnaryInterceptor(interceptor.DefaultServerUnaryInterceptors(&logger)...),
        grpc.ChainStreamInterceptor(interceptor.DefaultServerStreamInterceptors(&logger)...),
    )

    // 2. Register service implementation
    pbInventory.RegisterInventoryServiceServer(server, &InventoryHandler{})

    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to listen on port 50051")
    }

    logger.Info().Msg("gRPC server listening on :50051")
    if err := server.Serve(lis); err != nil {
        logger.Fatal().Err(err).Msg("gRPC server stopped")
    }
}
```

### Accessing User Identity & Request ID in Server Handlers:

```go
func (h *InventoryHandler) ReserveTickets(ctx context.Context, req *pbInventory.ReserveTicketsRequest) (*pbInventory.ReserveTicketsResponse, error) {
    // 1. Get Request ID
    reqID := grpcContext.GetRequestID(ctx)

    // 2. Get Authenticated User UUID
    userUUID, ok := grpcContext.GetUserID(ctx)
    if !ok {
        return nil, grpcErrors.Unauthenticated("user identity missing")
    }

    // 3. Get User Roles
    roles := grpcContext.GetUserRoles(ctx)

    // Business logic...
    if !hasStock {
        return nil, grpcErrors.FailedPrecondition("selected seats are no longer available")
    }

    return &pbInventory.ReserveTicketsResponse{
        ReservationId: "res-12345",
        Status:        "RESERVED",
    }, nil
}
```

---

## 3. Client Implementation Example

To call a gRPC microservice from another service (e.g., from `order-service`):

### 3.1. Create Client Connection:
```go
import (
    "time"
    "github.com/rs/zerolog"
    grpcClient "github.com/TranVinhHien/ticket-date/be/common/grpc/client"
    pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
)

conn, err := grpcClient.NewClientConn(
    "inventory-service:50051",
    grpcClient.WithTimeout(5*time.Second),
    grpcClient.WithLogger(&logger),
    grpcClient.WithInsecure(), // Use WithTransportCredentials for TLS in production
)
if err != nil {
    logger.Fatal().Err(err).Msg("failed to connect to inventory-service")
}
defer conn.Close()

inventoryClient := pbInventory.NewInventoryServiceClient(conn)
```

### 3.2. Call gRPC from Gin HTTP Handler (with Context Propagation):
```go
import (
    "github.com/gin-gonic/gin"
    grpcContext "github.com/TranVinhHien/ticket-date/be/common/grpc/context"
    grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
    "github.com/TranVinhHien/ticket-date/be/common/response"
)

func CreateOrderHandler(c *gin.Context) {
    // Automatically forwards X-Request-ID, X-User-ID, X-User-Roles from HTTP request
    // to the downstream gRPC service via outgoing metadata
    ctx := grpcContext.FromGinContext(c)

    res, err := inventoryClient.ReserveTickets(ctx, &pbInventory.ReserveTicketsRequest{
        OrderId: "order-1",
        EventId: "event-99",
        SeatIds: []string{"A1", "A2"},
    })

    if err != nil {
        httpStatus := grpcErrors.ToHTTPStatus(err)
        c.JSON(httpStatus, response.Error(grpcErrors.Message(err)))
        return
    }

    response.Success(c, res)
}
```

---

## 4. Error Mapping (`grpc/errors`)

Standard error constructors mapping to gRPC status codes:

| Function | gRPC Code | HTTP Status |
| :--- | :--- | :--- |
| `grpcErrors.InvalidArgument(...)` | `InvalidArgument` | `400 Bad Request` |
| `grpcErrors.Unauthenticated(...)` | `Unauthenticated` | `401 Unauthorized` |
| `grpcErrors.PermissionDenied(...)` | `PermissionDenied` | `403 Forbidden` |
| `grpcErrors.NotFound(...)` | `NotFound` | `404 Not Found` |
| `grpcErrors.AlreadyExists(...)` | `AlreadyExists` | `409 Conflict` |
| `grpcErrors.FailedPrecondition(...)`| `FailedPrecondition`| `412 Precondition Failed` |
| `grpcErrors.Unavailable(...)` | `Unavailable` | `503 Service Unavailable` |
| `grpcErrors.Internal(...)` | `Internal` | `500 Internal Server Error` |

```go
// Map any gRPC error to an HTTP status code for REST responses:
status := grpcErrors.ToHTTPStatus(err)
```

---

## 5. Protobuf Helpers (`grpc/helper`)

### Dynamic JSON / StructPB:
```go
import grpcHelper "github.com/TranVinhHien/ticket-date/be/common/grpc/helper"

// Map -> *structpb.Struct
st, err := grpcHelper.ConvertMapToStructPB(myMap)

// *structpb.Struct -> Map
m := grpcHelper.ConvertStructPBToMap(st)
```

### Timestamps:
```go
// time.Time -> *timestamppb.Timestamp
ts := grpcHelper.TimeToTimestamp(time.Now())

// *timestamppb.Timestamp -> time.Time
t := grpcHelper.TimestampToTime(ts)
```

---

## 6. Protobuf Compilation

To recompile `.proto` files:
```bash
make gen-proto
```
