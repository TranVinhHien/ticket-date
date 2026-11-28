# k6 Load Test – Luồng Đặt Vé

## Cấu trúc thư mục

```
k6/
├── ticket-booking.test.js      # Script k6 chính
├── setup/
│   ├── init-data.js            # Script khởi tạo dữ liệu (Node.js)
│   └── test-data.json          # Dữ liệu sinh ra (tự động tạo khi chạy init)
├── config/
│   ├── env.example.sh          # Mẫu biến môi trường (bash/zsh)
│   └── env.example.ps1         # Mẫu biến môi trường (PowerShell)
└── results/
    └── summary.json            # Kết quả test (tự động tạo)
```

---

## Điều kiện tiên quyết

| Tool | Version | Mục đích |
|---|---|---|
| **Node.js** | ≥ 18 | Chạy script khởi tạo dữ liệu |
| **k6** | ≥ 0.47 | Chạy load test |

Cài k6:
```bash
# Windows (Chocolatey)
choco install k6

# Windows (winget)
winget install k6

# macOS
brew install k6
```

---

## Quy trình chạy test

### Bước 1 – Chuẩn bị môi trường

**PowerShell:**
```powershell
# Sao chép và chỉnh sửa file cấu hình
copy k6\config\env.example.ps1 k6\config\env.ps1

# Chỉnh sửa URL và thông số trong env.ps1, sau đó load:
. .\k6\config\env.ps1
```

**Bash/Zsh (Linux/macOS):**
```bash
cp k6/config/env.example.sh k6/config/env.sh
# Chỉnh sửa URL và thông số
source k6/config/env.sh
```

### Bước 2 – Khởi tạo dữ liệu (chạy 1 lần trước khi test)

```bash
node k6/setup/init-data.js
```

Script sẽ:
1. Đăng ký + login admin user
2. Tạo organizer, venue
3. Tạo `NUM_EVENTS` event (mỗi event có 2 ticket_types: VIP và GA)
4. Tạo inventory `INVENTORY_QTY` vé/ticket_type
5. Tạo `NUM_USERS` test user (đăng ký + login để lấy token)
6. Lưu tất cả vào `k6/setup/test-data.json`

> **Lưu ý:** Chỉ cần chạy init một lần. Có thể chạy lại load test nhiều lần mà không cần tạo lại dữ liệu (trừ khi inventory cạn kiệt).

---

## Bước 3 – Chạy k6 load test

### Chế độ 1: Fixed VUs (số lượng VU cố định)

```bash
k6 run k6/ticket-booking.test.js \
  --env TEST_DATA_FILE=k6/setup/test-data.json \
  --env TEST_MODE=fixed_vus \
  --env VUS=10 \
  --env RAMP_UP=10s \
  --env DURATION=60s \
  --env THINK_TIME=1
```

### Chế độ 2: Fixed Iterations (số lượng iteration cố định)

```bash
k6 run k6/ticket-booking.test.js \
  --env TEST_DATA_FILE=k6/setup/test-data.json \
  --env TEST_MODE=fixed_iterations \
  --env VUS=10 \
  --env ITERATIONS=100 \
  --env DURATION=5m
```

---

## Ví dụ chạy với các mức concurrency khác nhau

### 10 VUs – Smoke test

```bash
k6 run k6/ticket-booking.test.js \
  --env TEST_DATA_FILE=k6/setup/test-data.json \
  --env VUS=10 \
  --env RAMP_UP=5s \
  --env DURATION=30s
```

> Khởi tạo dữ liệu: `NUM_USERS=10`

### 50 VUs – Load test

```bash
k6 run k6/ticket-booking.test.js \
  --env TEST_DATA_FILE=k6/setup/test-data.json \
  --env VUS=50 \
  --env RAMP_UP=20s \
  --env DURATION=3m
```

> Khởi tạo dữ liệu: `NUM_USERS=50`, `INVENTORY_QTY=50000`


```bash
k6 run ticket-booking.test.js `1m55s
>>   --env TEST_DATA_FILE=setup/test-data.json `tp_req_duration, order_success_rate, step_create_order_ms, step_get_by_slug_ms, step_list_events
>>   --env VUS=1000 `_ms' have been crossed
>>   --env DURATION=60sct\ticket-date\k6>
```


### 100 VUs – Stress test

```bash
k6 run k6/ticket-booking.test.js \
  --env TEST_DATA_FILE=k6/setup/test-data.json \
  --env VUS=100 \
  --env RAMP_UP=30s \
  --env DURATION=5m
```

> Khởi tạo dữ liệu: `NUM_USERS=100`, `INVENTORY_QTY=200000`

### 500 VUs – Peak test

```bash
k6 run k6/ticket-booking.test.js \
  --env TEST_DATA_FILE=k6/setup/test-data.json \
  --env VUS=500 \
  --env RAMP_UP=60s \
  --env DURATION=5m \
  --env THINK_TIME=0.5
```

> Khởi tạo dữ liệu: `NUM_USERS=500`, `INVENTORY_QTY=1000000`, `NUM_EVENTS=5`

---

## Các biến môi trường

### Init Data (`init-data.js`)

| Biến | Mặc định | Mô tả |
|---|---|---|
| `BASE_URL` | `http://localhost:8000` | URL auth và user service |
| `EVENT_URL` | `= BASE_URL` | URL event service |
| `INVENTORY_URL` | `= BASE_URL` | URL inventory service |
| `ORDER_URL` | `= BASE_URL` | URL order service |
| `ADMIN_EMAIL` | auto-generate | Email admin để tạo dữ liệu |
| `ADMIN_PASSWORD` | `Admin@123456` | Password admin |
| `NUM_EVENTS` | `1` | Số event cần tạo |
| `NUM_USERS` | `10` | Số test user (nên >= VUS) |
| `INVENTORY_QTY` | `100000` | Vé/ticket_type |

### k6 Test (`ticket-booking.test.js`)

| Biến | Mặc định | Mô tả |
|---|---|---|
| `TEST_DATA_FILE` | `k6/setup/test-data.json` | Đường dẫn file dữ liệu |
| `TEST_MODE` | `fixed_vus` | `fixed_vus` hoặc `fixed_iterations` |
| `VUS` | `10` | Số Virtual Users đồng thời |
| `RAMP_UP` | `10s` | Thời gian tăng dần VU |
| `DURATION` | `60s` | Thời gian chạy test |
| `ITERATIONS` | `100` | Tổng iteration (fixed_iterations mode) |
| `THINK_TIME` | `1` | Giây nghỉ giữa các bước (0 = không nghỉ) |

---

## Luồng nghiệp vụ (10 API calls / iteration)

```
VU (iteration)
 │
 ├─[1]  GET  /api/v1/events?status=PUBLISHED          → Danh sách event
 ├─[2]  GET  /api/v1/events/slug/{slug}               → Xem event theo slug
 ├─[3]  GET  /api/v1/events/{event_id}                → Xem event theo ID
 ├─[4]  GET  /api/v1/inventories/stock?ticket_type_ids=... → Kiểm tra tồn kho (query)
 ├─[5]  POST /api/v1/inventories/stock                → Xác nhận tồn kho (body)
 ├─[6]  POST /api/v1/orders                           → Đặt vé VIP (tạo order)
 ├─[7]  POST /api/v1/inventories/stock                → Verify stock giảm sau đặt vé
 ├─[8]  GET  /api/v1/users/me                         → Lấy thông tin user
 ├─[9]  POST /api/v1/inventories/stock                → Kiểm tra tồn kho GA
 └─[10] POST /api/v1/orders                           → Đặt vé GA (tạo order)
```

**Tính toán API calls:**

| Thông số | Công thức | Ví dụ (10 VUs × 60s) |
|---|---|---|
| 1 iteration | 10 API calls | 10 |
| N iterations | N × 10 | 1000 iterations → 10.000 calls |
| Throughput | VUS × (10 / avg_iter_time) | Phụ thuộc response time |

---

## Metrics và Thresholds

### Custom Metrics

| Metric | Loại | Mô tả |
|---|---|---|
| `orders_created_total` | Counter | Tổng số order tạo thành công |
| `orders_failed_total` | Counter | Tổng số order thất bại |
| `business_errors_total` | Counter | Lỗi nghiệp vụ (HTTP 2xx nhưng business fail) |
| `api_errors_total` | Counter | Lỗi HTTP (4xx, 5xx) |
| `error_rate` | Rate | Tỷ lệ lỗi tổng hợp |
| `order_success_rate` | Rate | Tỷ lệ đặt vé thành công |
| `step_list_events_ms` | Trend | Response time bước list events |
| `step_get_by_slug_ms` | Trend | Response time bước get by slug |
| `step_get_by_id_ms` | Trend | Response time bước get by ID |
| `step_stock_query_ms` | Trend | Response time kiểm tra stock |
| `step_create_order_ms` | Trend | Response time tạo order |

### Thresholds mặc định

| Threshold | Điều kiện |
|---|---|
| `error_rate` | < 5% |
| `order_success_rate` | > 90% |
| `http_req_duration p(95)` | < 3000ms |
| `step_create_order_ms p(95)` | < 2000ms |
| `step_list_events_ms p(95)` | < 1000ms |

---

## Phân biệt loại lỗi

```
Lỗi HTTP (api_errors_total):
  - HTTP 4xx: Request sai, unauthorized, not found
  - HTTP 5xx: Server error
  - Network error: Connection refused, timeout

Lỗi Nghiệp vụ (business_errors_total):
  - HTTP 200/201 nhưng body.code != "200"
  - Ví dụ: out of stock, order duplicate, payment failed
  - Inventory check fail (inventoryCheckFail counter)
```

---

## Lưu ý quan trọng

> [!IMPORTANT]
> `NUM_USERS` nên **lớn hơn hoặc bằng** `VUS`. Mỗi VU dùng 1 user xoay vòng; nếu `NUM_USERS < VUS`, nhiều VU sẽ chia sẻ cùng 1 token có thể gây race condition ở session.

> [!NOTE]
> Token trong `test-data.json` có thể hết hạn theo thời gian. Nếu test bị lỗi 401, hãy chạy lại `init-data.js` để refresh token.

> [!TIP]
> Để tránh hết inventory khi chạy nhiều lần, hãy đặt `INVENTORY_QTY` đủ lớn (ví dụ: `VUS × DURATION_seconds × 2 / avg_iter_seconds`). Hoặc dùng API Update Batch Stock để nạp thêm vé.
k6 run k6/ticket-booking.test.js: Chạy script load test kịch bản đặt vé (10 bước API calls mỗi vòng lặp).

  
--env TEST_DATA_FILE=k6/setup/test-data.json: Đường dẫn tới file JSON chứa dữ liệu test (token user, event_id, ticket_type_id, ...) được load vào SharedArray để dùng chung cho các Virtual Users. Mặc định là k6/setup/test-data.json.

  
--env VUS=500: Số lượng Virtual Users (người dùng ảo chạy đồng thời tối đa) đạt được trong giai đoạn tải chính. Mặc định là 10.

  
--env RAMP_UP=60s: Thời gian tăng dần số lượng VUs từ 0 lên 500 (stage 1 của executor ramping-vus) để hệ thống làm nóng và tăng tải từ từ thay vì dồn 500 user cùng lúc. Mặc định là 10s.

  
--env DURATION=5m: Thời gian duy trì tải ổn định ở mức 500 VUs (stage 2, ngay sau khi RAMP_UP kết thúc). Sau 5 phút này, script tự động có thêm 10s giảm tải (target: 0) ở cuối bài test. Mặc định là 60s.

  
--env THINK_TIME=0.5: Thời gian nghỉ (tính bằng giây, ở đây là 0.5s) mô phỏng khoảng dừng thao tác của người dùng thật giữa các bước gọi API (sleep(THINK_TIME)). Mặc định là 1 giây.

  k6 run k6/ticket-booking.test.js --env TEST_DATA_FILE=setup/test-data.json --env VUS=1000 --env RAMP_UP=5s --env DURATION=90s --env THINK_TIME=0.5