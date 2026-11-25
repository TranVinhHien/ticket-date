# =============================================================================
# k6 Load Test – Environment Variables (PowerShell)
# Sao chép thành env.ps1 và chỉnh sửa theo môi trường của bạn.
# =============================================================================

# ── API Endpoints ─────────────────────────────────────────────────────────────
$env:BASE_URL       = "http://localhost:8000"
$env:EVENT_URL      = "http://localhost:8000"
$env:INVENTORY_URL  = "http://localhost:8000"
$env:ORDER_URL      = "http://localhost:8000"

# ── Init Data Config ──────────────────────────────────────────────────────────
$env:ADMIN_EMAIL    = "admin@gmail.com"             # Để trống = tự generate
$env:ADMIN_PASSWORD = "123456"
$env:NUM_EVENTS     = "1"
$env:NUM_USERS      = "100"
$env:INVENTORY_QTY  = "1000"

# ── k6 Test Config ────────────────────────────────────────────────────────────
$env:TEST_MODE      = "fixed_vus"   # "fixed_vus" | "fixed_iterations"
$env:VUS            = "10"
$env:RAMP_UP        = "10s"
$env:DURATION       = "60s"
$env:ITERATIONS     = "100"
$env:THINK_TIME     = "1"
$env:TEST_DATA_FILE = "k6/setup/test-data.json"
