# K6 Performance & Load Test cho API Get Current Stock

Script k6 tại file `get_stock_perf.js` dùng để đo lường hiệu năng, thông lượng (RPS - Requests Per Second), độ trễ (latency p95, p99) và kiểm thử khả năng chịu tải của API:
```bash
curl --request GET 'http://localhost:8888/api/v1/inventories/stock?ticket_type_ids=0892eb9f-8d38-4b08-b134-c5de55b9c6be'
```

---

## 1. Cài đặt K6 trên Windows

Nếu máy bạn chưa có `k6`, bạn có thể cài đặt bằng một trong các cách sau:

### Cách 1: Bằng Windows Package Manager (Khuyên dùng)
```powershell
winget install k6 --source winget
```

### Cách 2: Bằng Chocolatey
```powershell
choco install k6
```

### Cách 3: Chạy trực tiếp qua Docker (Không cần cài k6 vào máy)
```powershell
docker run --rm -i --net="host" grafana/k6 run - < k6/get_stock_perf.js
```

---

## 2. Các Lệnh Chạy Kiểm Thử

### 2.1. Chạy mặc định (Tăng dần lên 100 người dùng trong 30s)
```powershell
k6 run k6/get_stock_perf.js
```

### 2.2. Tùy chỉnh số người đồng thời (VUs) và thời gian
Bạn có thể truyền các biến môi trường qua cờ `-e`:

- **Test 500 người dùng đồng thời tăng dần trong 1 phút**:
  ```powershell
  k6 run -e TARGET_VUS=500 -e DURATION=1m -e RAMP_TIME=15s k6/get_stock_perf.js
  ```

- **Test 1.000 người dùng đồng thời (Stress Test)**:
  ```powershell
  k6 run -e TARGET_VUS=1000 -e DURATION=45s k6/get_stock_perf.js
  ```

- **Test tải cố định ngay từ đầu (Constant Load - không tăng dần)**:
  ```powershell
  k6 run -e TARGET_VUS=200 -e TEST_MODE=constant -e DURATION=30s k6/get_stock_perf.js
  ```

- **Đổi ID vé hoặc địa chỉ server khác**:
  ```powershell
  k6 run -e BASE_URL=http://localhost:8888 -e TICKET_TYPE_ID=your-ticket-uuid-here k6/get_stock_perf.js
  ```

---

## 3. Bảng Tham Số Có Thể Cấu Hình

| Biến (`-e`) | Mặc định | Ý nghĩa |
|---|---|---|
| `TARGET_VUS` | `100` | Số lượng người dùng ảo đồng thời tối đa (Virtual Users). |
| `DURATION` | `30s` | Thời gian duy trì ở mức tải đỉnh. |
| `RAMP_TIME` | `10s` | Thời gian tăng tải (ramp up) và hạ tải (ramp down). |
| `TEST_MODE` | `ramp` | `ramp` (tăng tải dần theo giai đoạn) hoặc `constant` (tải cố định). |
| `BASE_URL` | `http://localhost:8888` | Địa chỉ Host của inventory-service. |
| `TICKET_TYPE_ID` | `0892eb9f-8d38-4b08-b134-c5de55b9c6be` | UUID của ticket type cần test. |
| `SLEEP_MS` | `0` | Thời gian nghỉ (ms) giữa các request của cùng 1 VU (để `0` nhằm ép throughput RPS tối đa). |

---

## 4. Báo Cáo Kết Quả Sau Khi Chạy Xong

Sau khi bài test kết thúc, script sẽ tự động xuất ra 3 định dạng báo cáo:

1. **Terminal Console**:
   In bảng tóm tắt chi tiết trực tiếp trên màn hình:
   - **Throughput (RPS)**: Số lượng request API phục vụ được trên 1 giây.
   - **Độ trễ**: Min, Max, Avg, P90, P95, P99.
   - **Tỉ lệ lỗi**: Phần trăm request bị fail hoặc timeout.
2. **Dashboard HTML**: File `k6/report.html` (Mở trực tiếp bằng trình duyệt Chrome/Edge để xem biểu đồ giao diện đẹp mắt).
3. **Dữ liệu thô JSON**: File `k6/summary.json` (dùng để lưu trữ hoặc tích hợp CI/CD).
