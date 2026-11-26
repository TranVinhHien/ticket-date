# Kafka & Asynchronous Event-Driven Architecture Guide

Tài liệu này giải thích chi tiết kiến trúc, flow xử lý, data contract và logic Kafka dùng trong hệ thống `ticket-date` (đặc biệt giữa `order-service` và `inventory-service`) để các Developer và Agent sau này có thể đọc và hiểu ngay.

---

## 1. Tổng quan Kiến trúc (High-Level Architecture)

Hệ thống sử dụng **Transactional Outbox Pattern** kết hợp với **Kafka Event-Driven Architecture**:

```
[Order Creation Flow]
Client POST /api/v1/orders
       │
       ▼
[order-service]
  1. Validate request, merge duplicate tickets
  2. gRPC GetTicketTypes -> event-service
  3. gRPC DeductStock (Redis atomic) -> inventory-service
  4. BEGIN DB Transaction (PostgreSQL):
       ├── INSERT INTO orders (...)
       ├── INSERT INTO order_items (...)
       └── INSERT INTO outbox_event (status=0 PENDING, event_type='OrderCreated', payload=...)
     COMMIT DB Transaction
       │
       ▼
[Outbox Relay Cron Job] (Every 2s)
  1. RecoverStaleSendingOutboxEvents (Status 1 SENDING > 120s -> 0 PENDING / 3 FAILED)
  2. ClaimPendingOutboxEvents (SELECT ... FOR UPDATE SKIP LOCKED -> status=1 SENDING)
  3. Publish to Kafka:
       - Destination Topic = ev.EventType ("OrderCreated")
       - If broker fails: Retry 3 times in-memory with exponential backoff (100ms, 200ms, 400ms).
       - If all 3 in-memory retries fail -> UPDATE outbox_event SET status=3 (FAILED) duy nhất 1 lần (chống nghẽn DB).
       - If success -> UPDATE outbox_event SET status=2 (SUCCESS).
       │
       ▼
 [Kafka Broker: Topic "OrderCreated"]
       │
       ▼
[inventory-service] (Consumer Group: "inventory-service-group")
  1. Partition Worker tuần tự hóa message theo partition.
  2. Handler: OrderCreatedConsumer
  3. BEGIN DB Transaction:
       ├── TryAcquireIdempotencyKey (token = "OrderCreated:<order_id>") -> Chống duplicate event
       ├── CreateInventoryTransaction (HOLD)
       ├── HoldTicketInventoryStockBatch (1 Batch UPDATE qua UNNEST - chống N+1 query)
       └── CreateInventoryTransactionItemBatch (1 Batch INSERT qua UNNEST)
     COMMIT DB Transaction
```

---

## 2. Shared Library: `be/common/kafka`

Toàn bộ logic Kafka dùng chung được đóng gói tại `be/common/kafka`:

### 2.1. Event Envelope (`envelope.go`)
Mọi message gửi lên Kafka đều được bọc trong một chuẩn chung `EventEnvelope`:
```json
{
  "event_id": "outbox-101",
  "event_type": "OrderCreated",
  "version": "v1",
  "source": "order-service",
  "occurred_at": "2026-11-26T12:00:00Z",
  "trace_id": "req-xyz-123",
  "data": { ... }
}
```
- Headers trên Kafka record: `x-event-id`, `x-event-type`, `x-source-service`, `x-trace-id`.
- Helper: `commonKafka.DecodeData[T](env)` giải mã payload generic an toàn.

### 2.2. Producer (`producer.go`)
- Dựa trên thư viện hiệu năng cao `twmb/franz-go`.
- Mặc định bật nén **LZ4**, idempotent write (`acks=all`), buffer batching `LingerMs=10ms`.
- `PublishRawSync(...)`: Nhận pre-serialized JSON từ outbox và publish đồng bộ để đảm bảo outbox relay job chỉ cập nhật DB khi Kafka đã ACK.

### 2.3. Consumer (`consumer.go`)
- Quản lý worker goroutine riêng theo từng `(topic, partition)`. Điều này đảm bảo:
  - **Thứ tự tuyệt đối** bên trong 1 partition.
  - **Xử lý song song** giữa các partition khác nhau.
- Tự động rebalance an toàn (`OnPartitionsRevoked` chờ worker xả xong message hiện tại rồi mới commit offset).
- Tích hợp **Dead Letter Queue (DLQ)**: nếu xử lý event thất bại sau `max_retries` (mặc định 3 lần), event được chuyển tiếp sang `<topic>.dlq` và commit offset để tránh tắc nghẽn partition.

---

## 3. Order Service: Outbox Relay Cron (`internal/cron`)

### 3.1. Topic theo EventType
- `targetTopic := ev.EventType`: Cho phép `order-service` phát nhiều loại event khác nhau (ví dụ: `OrderCreated`, `OrderCancelled`, `OrderRefunded`) qua cùng 1 bảng outbox mà không bị gò bó vào 1 topic duy nhất.

### 3.2. Cơ chế Chống nghẽn Database (Throttling DB Updates khi Kafka sự cố)
- **Vấn đề đã xử lý**: Nếu Kafka bị lỗi hoặc sập cả cụm, việc update `status` & `retry_count` về DB sau mỗi lần lỗi sẽ làm hàng nghìn row cùng lúc ghi đè DB, gây nghẽn connection pool và sập cả database.
- **Giải pháp**:
  - Relay job thử publish lại **ngay trong bộ nhớ (in-memory)** tối đa 3 lần (`maxRetries`) kèm exponential backoff.
  - Chỉ khi cả 3 lần thử đều thất bại, job mới gọi `MarkOutboxEventFailed` để ghi nhận `status = 3 (FAILED)` **duy nhất 1 lần vào DB**.

---

## 4. Inventory Service: Consumer & Batch DB Optimization

### 4.1. Consumer Registration
File: `internal/consumers/order_created_consumer.go`
- Đăng ký lắng nghe topic `OrderCreated` với handler `NewOrderCreatedHandler(invService)`.

### 4.2. Khử N+1 Query trong `HandleOrderCreated`
- Trước đây: Mỗi ticket type trong order phải chạy 2 query riêng lẻ trong vòng lặp (`HoldTicketInventoryStock` + `CreateInventoryTransactionItem`), dẫn đến `2N` round-trips tới PostgreSQL.
- Hiện tại: Tối ưu hoá thành **2 batch queries dùng PostgreSQL UNNEST**:
  1. `HoldTicketInventoryStockBatch`: Cập nhật `held_quantity` cho toàn bộ danh sách ticket type trong 1 query duy nhất, sắp xếp theo `ticket_type_id` để triệt tiêu deadlock:
     ```sql
     WITH requested AS (
         SELECT unnest($1::uuid[]) AS ticket_type_id, unnest($2::int[]) AS qty ORDER BY 1
     )
     UPDATE ticket_inventories ti
     SET held_quantity = ti.held_quantity + r.qty, updated_at = NOW()
     FROM requested r
     WHERE ti.ticket_type_id = r.ticket_type_id
       AND ti.deleted_at IS NULL
       AND ti.held_quantity + ti.sold_quantity + r.qty <= ti.total_quantity
     RETURNING ti.*;
     ```
  2. `CreateInventoryTransactionItemBatch`: Insert toàn bộ line item vào `inventory_transaction_items` trong 1 query `INSERT ... SELECT unnest(...)`.
- Kiểm tra tính toàn vẹn: Nếu số lượng inventory cập nhật được khác `len(sortedIDs)`, hệ thống lập tức rollback và báo lỗi thiếu hàng/thiếu vé.

---

## 5. Quy chuẩn Thiết lập Môi trường (Environment Setup)

### Chạy Kafka cục bộ qua Docker (Ubuntu / Linux)
```bash
docker run -d --name ticket-date-kafka -p 9092:9092 \
  -e KAFKA_NODE_ID=1 \
  -e KAFKA_PROCESS_ROLES=broker,controller \
  -e KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1 \
  -e KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0 \
  -e KAFKA_AUTO_CREATE_TOPICS_ENABLE=true \
  apache/kafka:latest
```

### Cấu hình `app.development.yaml`
- **order-service**:
  ```yaml
  kafka:
    brokers: ["localhost:9092"]
    client_id: "order-service"
    compression: "lz4"
    allow_auto_topic: true
  cron:
    enabled: true
    pending_orders:
      enabled: true
      interval: "1m"
      batch: 100
    outbox_relay:
      enabled: true
      interval: "2s"
      batch: 100
  ```
- **inventory-service**:
  ```yaml
  kafka:
    brokers: ["localhost:9092"]
    client_id: "inventory-service"
    group_id: "inventory-service-group"
    topics: ["OrderCreated"]
    max_retries: 3
    retry_backoff: "500ms"
    dlq_enabled: true
    dlq_suffix: ".dlq"
  ```
