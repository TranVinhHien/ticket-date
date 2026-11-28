/**
 * ticket-booking.test.js
 * ======================
 * k6 Load Test: Luồng đặt vé sự kiện
 *
 * Luồng nghiệp vụ (10 API calls / iteration):
 *   [1]  GET  /api/v1/events?page=1&limit=10&status=PUBLISHED  → List Events
 *   [2]  GET  /api/v1/events/slug/{slug}                       → Get Event by Slug
 *   [3]  GET  /api/v1/events/{event_id}                        → Get Event by ID
 *   [4]  GET  /api/v1/inventories/stock?ticket_type_ids=...    → Get Stock (Query)
 *   [5]  POST /api/v1/inventories/stock                        → Get Stock (Body) – xác nhận tồn kho
 *   [6]  POST /api/v1/orders                                   → Create Order (lần 1)
 *   [7]  POST /api/v1/inventories/stock                        → Kiểm tra stock sau khi đặt
 *   [8]  GET  /api/v1/users/me                                 → Get Profile (xác nhận user)
 *   [9]  POST /api/v1/inventories/stock                        → Verify stock giảm
 *   [10] POST /api/v1/orders                                   → Create Order (lần 2 – ticket_type thứ hai)
 *
 * Cấu hình qua environment variables (xem k6/config/env.js hoặc truyền --env):
 *   BASE_URL, EVENT_URL, INVENTORY_URL, ORDER_URL
 *   VUS, DURATION, ITERATIONS, RAMP_UP, THINK_TIME
 *   TEST_MODE: "fixed_vus" | "fixed_iterations"
 *   TEST_DATA_FILE: đường dẫn tới file JSON sinh ra bởi init-data.js
 *
 * Chạy:
 *   k6 run k6/ticket-booking.test.js \
 *     --env TEST_DATA_FILE=k6/setup/test-data.json \
 *     --env VUS=10 \
 *     --env DURATION=60s
 */

import http from "k6/http";
import { check, group, sleep, fail } from "k6";
import { Counter, Rate, Trend } from "k6/metrics";
import { SharedArray } from "k6/data";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

// ─── Custom Metrics ───────────────────────────────────────────────────────────
const orderCreated = new Counter("orders_created_total");
const orderFailed = new Counter("orders_failed_total");
const inventoryCheckFail = new Counter("inventory_check_failures");
const businessErrors = new Counter("business_errors_total");
const apiErrors = new Counter("api_errors_total");

const listEventsDuration = new Trend("step_list_events_ms", true);
const getBySlugDuration = new Trend("step_get_by_slug_ms", true);
const getByIdDuration = new Trend("step_get_by_id_ms", true);
const stockQueryDuration = new Trend("step_stock_query_ms", true);
const stockBodyDuration = new Trend("step_stock_body_ms", true);
const createOrderDuration = new Trend("step_create_order_ms", true);
const getProfileDuration = new Trend("step_get_profile_ms", true);

const errorRate = new Rate("error_rate");
const orderSuccessRate = new Rate("order_success_rate");

// ─── Load Test Data ───────────────────────────────────────────────────────────
// TEST_DATA_FILE phải trỏ tới file JSON sinh bởi init-data.js
const TEST_DATA_FILE = __ENV.TEST_DATA_FILE || "setup/test-data.json";
const REPORT_DIR = __ENV.REPORT_DIR || "k6/results";

const testData = new SharedArray("testData", function () {
  return [JSON.parse(open(TEST_DATA_FILE))];
});

// ─── Config ───────────────────────────────────────────────────────────────────
const TEST_MODE = __ENV.TEST_MODE || "fixed_vus";
const VUS = parseInt(__ENV.VUS || "10", 10);
const DURATION = __ENV.DURATION || "60s";
const ITERATIONS = parseInt(__ENV.ITERATIONS || "100", 10);
const RAMP_UP = __ENV.RAMP_UP || "10s";
const THINK_TIME = parseFloat(__ENV.THINK_TIME || "1");

// ─── k6 Options ───────────────────────────────────────────────────────────────
export const options = buildOptions();

function buildOptions() {
  const thresholds = {
    // Tỷ lệ lỗi HTTP dưới 5%
    error_rate: ["rate<0.05"],
    // Tỷ lệ đặt vé thành công trên 90%
    order_success_rate: ["rate>0.9"],
    // P95 toàn bộ http request dưới 3s
    http_req_duration: ["p(95)<3000"],
    // Từng bước quan trọng
    step_create_order_ms: ["p(95)<2000"],
    step_list_events_ms: ["p(95)<1000"],
    step_get_by_slug_ms: ["p(95)<1000"],
    step_stock_query_ms: ["p(95)<1000"],
  };

  if (TEST_MODE === "fixed_iterations") {
    return {
      scenarios: {
        ticket_booking: {
          executor: "shared-iterations",
          vus: VUS,
          iterations: ITERATIONS,
          maxDuration: DURATION,
        },
      },
      thresholds,
    };
  }

  // Default: fixed_vus với ramp-up
  return {
    scenarios: {
      ticket_booking: {
        executor: "ramping-vus",
        startVUs: 0,
        stages: [
          { duration: RAMP_UP, target: VUS },
          { duration: DURATION, target: VUS },
          { duration: "10s", target: 0 },
        ],
      },
    },
    thresholds,
  };
}

// ─── Helper Functions ─────────────────────────────────────────────────────────

/**
 * Lấy dữ liệu test cho VU hiện tại.
 * Mỗi VU lần lượt dùng user và event từ pool đã tạo sẵn.
 */
function getVuData() {
  const data = testData[0];
  const vuIndex = __VU - 1;

  // Chọn user xoay vòng theo VU
  const user = data.users[vuIndex % data.users.length];

  // Chọn event xoay vòng theo VU (tránh nhiều VU cùng đặt 1 event nếu có nhiều event)
  const event = data.events[vuIndex % data.events.length];

  return { data, user, event };
}

/**
 * Lấy common headers có Authorization
 */
function authHeaders(token) {
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };
}

/**
 * Parse response body, trả về { ok, code, data, raw }
 * ok = true nếu HTTP 2xx VÀ body.code == "200" (hoặc "201")
 */
function parseResponse(res, label) {
  const isHttp2xx = res.status >= 200 && res.status < 300;

  let body;
  try {
    body = JSON.parse(res.body);
  } catch {
    body = null;
  }

  const bodyCode = body?.code;
  const isBodyOk = bodyCode === "200" || bodyCode === "201" || bodyCode === undefined;

  const ok = isHttp2xx && isBodyOk;

  if (!isHttp2xx) {
    apiErrors.add(1);
    errorRate.add(1);
  } else if (!isBodyOk) {
    businessErrors.add(1);
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }

  return { ok, httpStatus: res.status, code: bodyCode, data: body?.data, raw: res.body };
}

// ─── Main VU Flow ─────────────────────────────────────────────────────────────
export default function () {
  const { data, user, event } = getVuData();

  const BASE_URL = data.base_url;
  const EVENT_URL = data.event_url;
  const INVENTORY_URL = data.inventory_url;
  const ORDER_URL = data.order_url;

  // Lấy token từ shared data (token đã được tạo trong init-data.js)
  const token = user.token;

  // Lấy ticket_types của event được chọn
  const ticketTypeVip = event.ticket_types[0]; // VIP
  const ticketTypeGa = event.ticket_types[1]; // General Admission
  if (!ticketTypeVip || !ticketTypeGa) {
    fail(`Event ${event.event_id} không có đủ ticket_types`);
  }

  // ── [1] List Events ──────────────────────────────────────────────────────
  group("01_list_events", function () {
    const res = http.get(
      `${EVENT_URL}/api/v1/events?page=1&limit=10&status=PUBLISHED`,
      { tags: { step: "list_events" } }
    );
    listEventsDuration.add(res.timings.duration);

    const r = parseResponse(res, "List Events");
    check(res, {
      "[1] List Events - HTTP 200": (r) => r.status === 200,
      "[1] List Events - has data": () => r.data !== null && r.data !== undefined,
    });
  });

  if (THINK_TIME > 0) sleep(THINK_TIME * 0.5);

  // ── [2] Get Event by Slug ────────────────────────────────────────────────
  group("02_get_event_by_slug", function () {
    const res = http.get(
      `${EVENT_URL}/api/v1/events/slug/${event.event_slug}`,
      { tags: { step: "get_by_slug" } }
    );
    getBySlugDuration.add(res.timings.duration);

    const r = parseResponse(res, "Get Event by Slug");
    check(res, {
      "[2] Get by Slug - HTTP 200": (r) => r.status === 200,
      "[2] Get by Slug - correct ID": () => r.data?.id === event.event_id,
    });
  });

  if (THINK_TIME > 0) sleep(THINK_TIME * 0.5);

  // ── [3] Get Event by ID ──────────────────────────────────────────────────
  group("03_get_event_by_id", function () {
    const res = http.get(
      `${EVENT_URL}/api/v1/events/${event.event_id}`,
      { tags: { step: "get_by_id" } }
    );
    getByIdDuration.add(res.timings.duration);

    const r = parseResponse(res, "Get Event by ID");
    check(res, {
      "[3] Get by ID - HTTP 200": (r) => r.status === 200,
      "[3] Get by ID - has ticket_types": () => Array.isArray(r.data?.ticket_types),
    });
  });

  if (THINK_TIME > 0) sleep(THINK_TIME);

  // ── [4] Get Current Stock (Query) ────────────────────────────────────────
  group("04_get_stock_query", function () {
    const idsParam = `${ticketTypeVip.id},${ticketTypeGa.id}`;
    const res = http.get(
      `${INVENTORY_URL}/api/v1/inventories/stock?ticket_type_ids=${idsParam}`,
      {
        headers: authHeaders(token),
        tags: { step: "stock_query" },
      }
    );
    stockQueryDuration.add(res.timings.duration);

    const r = parseResponse(res, "Get Stock (Query)");
    const stockOk = check(res, {
      "[4] Stock Query - HTTP 200": (r) => r.status === 200,
      "[4] Stock Query - has items": () => r.data !== null && r.data !== undefined,
    });

    if (!stockOk) inventoryCheckFail.add(1);
  });

  if (THINK_TIME > 0) sleep(THINK_TIME * 0.3);

  // ── [5] Get Current Stock (GET) – xác nhận tồn kho VIP trước khi đặt ────
  // NOTE: Backend chỉ hỗ trợ GET cho endpoint này (POST trả về lỗi).
  // Dùng GET giống bước 4 nhưng chỉ query riêng VIP ticket type.
  group("05_get_stock_vip", function () {
    const res = http.get(
      `${INVENTORY_URL}/api/v1/inventories/stock?ticket_type_ids=${ticketTypeVip.id}`,
      {
        headers: authHeaders(token),
        tags: { step: "stock_vip" },
      }
    );
    stockQueryDuration.add(res.timings.duration);

    const r = parseResponse(res, "Get Stock VIP");
    const stockOk = check(res, {
      "[5] Stock VIP - HTTP 200": (r) => r.status === 200,
      "[5] Stock VIP - has items": () => r.data !== null && r.data !== undefined,
    });

    if (!stockOk) inventoryCheckFail.add(1);
  });

  if (THINK_TIME > 0) sleep(THINK_TIME);

  // ── [6] Create Order (VIP ticket) ───────────────────────────────────────
  let orderVipCreated = false;
  group("06_create_order_vip", function () {
    const idempotencyKey = uuidv4();
    const res = http.post(
      `${ORDER_URL}/api/v1/orders`,
      JSON.stringify({
        idempotency_key: idempotencyKey,
        items: [
          {
            ticket_type_id: ticketTypeVip.id,
            quantity: 1,
          },
        ],
      }),
      {
        headers: authHeaders(token),
        tags: { step: "create_order" },
      }
    );
    createOrderDuration.add(res.timings.duration);

    const r = parseResponse(res, "Create Order VIP");
    orderVipCreated = check(res, {
      "[6] Create Order VIP - HTTP 2xx": (r) => r.status >= 200 && r.status < 300,
      "[6] Create Order VIP - has order_id": () => !!r.data?.id,
    });

    if (orderVipCreated) {
      orderCreated.add(1);
      orderSuccessRate.add(1);
    } else {
      orderFailed.add(1);
      orderSuccessRate.add(0);

      // Phân loại lỗi nghiệp vụ vs lỗi HTTP
      if (res.status >= 200 && res.status < 300) {
        // HTTP OK nhưng body báo lỗi nghiệp vụ (vd: hết vé)
        businessErrors.add(1);
      }
    }
  });

  if (THINK_TIME > 0) sleep(THINK_TIME * 0.5);

  // ── [7] Get Profile ──────────────────────────────────────────────────────
  // group("07_get_profile", function () {
  //   const res = http.get(
  //     `${BASE_URL}/api/v1/users/me`,
  //     {
  //       headers: { Authorization: `Bearer ${token}` },
  //       tags: { step: "get_profile" },
  //     }
  //   );
  //   getProfileDuration.add(res.timings.duration);

  //   const r = parseResponse(res, "Get Profile");
  //   check(res, {
  //     "[7] Get Profile - HTTP 200":    (r) => r.status === 200,
  //     "[7] Get Profile - has user_id": () => !!r.data?.id,
  //   });
  // });

  if (THINK_TIME > 0) sleep(THINK_TIME * 0.5);

  // ── [8] Create Order (GA ticket) ────────────────────────────────────────
  group("08_create_order_ga", function () {
    const idempotencyKey = uuidv4();
    const res = http.post(
      `${ORDER_URL}/api/v1/orders`,
      JSON.stringify({
        idempotency_key: idempotencyKey,
        items: [
          {
            ticket_type_id: ticketTypeGa.id,
            quantity: 1,
          },
        ],
      }),
      {
        headers: authHeaders(token),
        tags: { step: "create_order_ga" },
      }
    );
    createOrderDuration.add(res.timings.duration);

    const r = parseResponse(res, "Create Order GA");
    const orderGaCreated = check(res, {
      "[8] Create Order GA - HTTP 2xx": (r) => r.status >= 200 && r.status < 300,
      "[8] Create Order GA - has order_id": () => !!r.data?.id,
    });

    if (orderGaCreated) {
      orderCreated.add(1);
      orderSuccessRate.add(1);
    } else {
      orderFailed.add(1);
      orderSuccessRate.add(0);

      if (res.status >= 200 && res.status < 300) {
        businessErrors.add(1);
      }
    }
  });

  // Think time cuối iteration
  if (THINK_TIME > 0) sleep(THINK_TIME);
}

// ─── Summary Report ───────────────────────────────────────────────────────────
export function handleSummary(data) {
  const metrics = data.metrics;

  // v(name) trả về đúng object .values – KHÔNG double-access
  function v(name) {
    return metrics[name]?.values ?? null;
  }

  function fmt(val, unit = "") {
    if (val === null || val === undefined || (typeof val === "number" && isNaN(val))) return "N/A";
    if (typeof val === "number") return `${val.toFixed(2)}${unit}`;
    return String(val);
  }

  // ── Đọc metrics theo đúng cấu trúc k6 data ────────────────────────────
  const totalIter = v("iterations")?.count ?? 0;
  const totalReqs = v("http_reqs")?.count ?? 0;
  const throughput = v("http_reqs")?.rate ?? 0;
  const reqDurAvg = v("http_req_duration")?.avg ?? 0;
  const reqDurMed = v("http_req_duration")?.med ?? 0;
  const reqDurMin = v("http_req_duration")?.min ?? 0;
  const reqDurMax = v("http_req_duration")?.max ?? 0;
  const reqDurP95 = v("http_req_duration")?.["p(95)"] ?? 0;
  const reqDurP99 = v("http_req_duration")?.["p(99)"] ?? 0;

  const ordersOk = v("orders_created_total")?.count ?? 0;
  const ordersFail = v("orders_failed_total")?.count ?? 0;
  const bizErrors = v("business_errors_total")?.count ?? 0;
  const httpErrors = v("api_errors_total")?.count ?? 0;
  const invFail = v("inventory_check_failures")?.count ?? 0;
  const errRate = v("error_rate")?.rate ?? 0;
  const orderSucRate = v("order_success_rate")?.rate ?? 0;

  const listP95 = v("step_list_events_ms")?.["p(95)"] ?? 0;
  const listAvg = v("step_list_events_ms")?.avg ?? 0;
  const slugP95 = v("step_get_by_slug_ms")?.["p(95)"] ?? 0;
  const idP95 = v("step_get_by_id_ms")?.["p(95)"] ?? 0;
  const stockQP95 = v("step_stock_query_ms")?.["p(95)"] ?? 0;
  const stockBP95 = v("step_stock_body_ms")?.["p(95)"] ?? 0;
  const orderP95 = v("step_create_order_ms")?.["p(95)"] ?? 0;
  const orderAvg = v("step_create_order_ms")?.avg ?? 0;
  const profileP95 = v("step_get_profile_ms")?.["p(95)"] ?? 0;

  const totalOrders = ordersOk + ordersFail;

  const p = (s) => s.padEnd(44);

  const summary = `
╔════════════════════════════════════════════════════════════════╗
║           k6 LOAD TEST REPORT – Luồng Đặt Vé                 ║
╠════════════════════════════════════════════════════════════════╣
║  Cấu hình                                                      ║
║    Mode        : ${p(TEST_MODE)}║
║    VUs         : ${p(String(VUS))}║
║    Duration    : ${p(String(DURATION))}║
╠════════════════════════════════════════════════════════════════╣
║  Volume & Throughput                                           ║
║    Iterations  : ${p(String(totalIter))}║
║    Total Reqs  : ${p(String(totalReqs))}║
║    Reqs/sec    : ${p(fmt(throughput, " rps"))}║
╠════════════════════════════════════════════════════════════════╣
║  Response Time (toàn bộ HTTP requests)                        ║
║    Min / Avg / Med : ${p(fmt(reqDurMin, " ms") + " / " + fmt(reqDurAvg, " ms") + " / " + fmt(reqDurMed, " ms"))}║
║    P95         : ${p(fmt(reqDurP95, " ms"))}║
║    P99         : ${p(fmt(reqDurP99, " ms"))}║
║    Max         : ${p(fmt(reqDurMax, " ms"))}║
╠════════════════════════════════════════════════════════════════╣
║  Response Time từng bước (P95)                                ║
║    [1] List Events   : ${p(fmt(listP95, " ms") + " avg:" + fmt(listAvg, " ms"))}║
║    [2] Get by Slug   : ${p(fmt(slugP95, " ms"))}║
║    [3] Get by ID     : ${p(fmt(idP95, " ms"))}║
║    [4] Stock Query   : ${p(fmt(stockQP95, " ms"))}║
║    [5,7,9] Stock Body: ${p(fmt(stockBP95, " ms"))}║
║    [6,10] Create Order: ${p(fmt(orderP95, " ms") + " avg:" + fmt(orderAvg, " ms"))}║
║    [8] Get Profile   : ${p(fmt(profileP95, " ms"))}║
╠════════════════════════════════════════════════════════════════╣
║  Lỗi                                                           ║
║    Error Rate  : ${p(fmt(errRate * 100, " %"))}║
║    HTTP Errors : ${p(String(httpErrors))}║
║    Biz Errors  : ${p(String(bizErrors) + " (HTTP 2xx nhưng business fail)")}║
║    Inv Fail    : ${p(String(invFail))}║
╠════════════════════════════════════════════════════════════════╣
║  Đặt Vé (Orders)                                              ║
║    Tổng order  : ${p(String(totalOrders))}║
║    Thành công  : ${p(String(ordersOk))}║
║    Thất bại    : ${p(String(ordersFail))}║
║    Success Rate: ${p(fmt(orderSucRate * 100, " %"))}║
╚════════════════════════════════════════════════════════════════╝
`;

  console.log(summary);

  // Timestamp để không ghi đè file cũ
  const ts = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);

  return {
    stdout: summary + "\n" + textSummary(data, { indent: "  ", enableColors: true }),
    [`${REPORT_DIR}/summary-${ts}.json`]: JSON.stringify(data, null, 2),
    [`${REPORT_DIR}/report-${ts}.html`]: htmlReport(data, { title: `Ticket Booking Load Test – ${ts}` }),
  };
}
