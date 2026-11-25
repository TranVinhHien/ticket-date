#!/bin/sh
# =============================================================================
# k6 Load Test – Environment Variables Configuration
# Sao chép thành env.sh và chỉnh sửa theo môi trường của bạn.
# =============================================================================

# ── API Endpoints ─────────────────────────────────────────────────────────────
# BASE_URL: URL gốc, dùng cho auth và user service
export BASE_URL="http://localhost:8000"

# Nếu các service chạy trên port khác nhau, set riêng:
export EVENT_URL="http://localhost:8000"
export INVENTORY_URL="http://localhost:8000"
export ORDER_URL="http://localhost:8000"

# ── Init Data Config ──────────────────────────────────────────────────────────
# Tài khoản admin để tạo organizer, venue, event
# Nếu để trống, script sẽ tự generate email ngẫu nhiên
export ADMIN_EMAIL=""
export ADMIN_PASSWORD="Admin@123456"

# Số event cần tạo (mỗi event có 2 ticket_types)
export NUM_EVENTS=1

# Số test user cần tạo (nên >= VUS để tránh nhiều VU dùng chung 1 token)
export NUM_USERS=10

# Số lượng vé mỗi ticket_type (đủ cho load test không bị out-of-stock)
export INVENTORY_QTY=100000

# ── k6 Test Config ────────────────────────────────────────────────────────────
# TEST_MODE: "fixed_vus" (default) | "fixed_iterations"
export TEST_MODE="fixed_vus"

# Số Virtual Users chạy đồng thời
export VUS=10

# Thời gian ramp-up (tăng dần VU lên đủ số)
export RAMP_UP="10s"

# Thời gian chạy test (chỉ áp dụng cho fixed_vus mode)
export DURATION="60s"

# Tổng số iteration (chỉ áp dụng cho fixed_iterations mode)
export ITERATIONS=100

# Think time (giây) giữa các bước trong 1 iteration (0 = không nghỉ)
export THINK_TIME=1

# Đường dẫn file dữ liệu sinh bởi init-data.js
export TEST_DATA_FILE="k6/setup/test-data.json"
