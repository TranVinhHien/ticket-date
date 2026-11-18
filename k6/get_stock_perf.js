import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// ============================================================================
// 1. CẤU HÌNH BIẾN MÔI TRƯỜNG (__ENV) CÓ THỂ ĐIỀU CHỈNH KHI CHẠY
// ============================================================================
// Ví dụ chạy:
//   k6 run -e TARGET_VUS=200 -e DURATION=30s k6/get_stock_perf.js
//   k6 run -e TARGET_VUS=500 -e TEST_MODE=constant k6/get_stock_perf.js
// ============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8888';
const TICKET_TYPE_ID = __ENV.TICKET_TYPE_ID || '0892eb9f-8d38-4b08-b134-c5de55b9c6be';

const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '100', 10); // Số lượng người đồng thời tối đa
const DURATION = __ENV.DURATION || '30s';                     // Thời gian duy trì mức tải đỉnh
const RAMP_TIME = __ENV.RAMP_TIME || '10s';                   // Thời gian tăng tải (ramp up / ramp down)
const TEST_MODE = __ENV.TEST_MODE || 'ramp';                  // 'ramp' (tăng dần) hoặc 'constant' (cố định ngay từ đầu)
const SLEEP_MS = parseFloat(__ENV.SLEEP_MS || '0');           // Nghỉ giữa các request (mặc định 0 để đo throughput max)

// Custom Metrics
const stockRequestsTotal = new Counter('stock_reqs_total');
const stockSuccessRate = new Rate('stock_success_rate');
const stockLatency = new Trend('stock_latency_ms', true);

// ============================================================================
// 2. KỊCH BẢN TẢI (SCENARIOS & STAGES)
// ============================================================================
export const options = (() => {
  // Mode 1: Cố định người dùng đồng thời (Constant VUs)
  if (TEST_MODE === 'constant') {
    return {
      summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
      scenarios: {
        constant_load: {
          executor: 'constant-vus',
          vus: TARGET_VUS,
          duration: DURATION,
        },
      },
      thresholds: {
        http_req_failed: ['rate<0.01'],              // Lỗi HTTP phải < 1%
        http_req_duration: ['p(95)<30', 'p(99)<80'], // 95% request < 30ms (do đã có Redis cache)
        stock_success_rate: ['rate>0.99'],           // Tỉ lệ thành công > 99%
      },
    };
  }

  // Mode 2 (Mặc định): Tăng dần người dùng (Ramping VUs)
  return {
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
    scenarios: {
      ramping_load: {
        executor: 'ramping-vus',
        startVUs: 1,
        stages: [
          { duration: RAMP_TIME, target: Math.floor(TARGET_VUS / 3) }, // Giai đoạn 1: Warm-up tăng lên 33%
          { duration: RAMP_TIME, target: TARGET_VUS },                 // Giai đoạn 2: Tăng lên đỉnh 100%
          { duration: DURATION, target: TARGET_VUS },                  // Giai đoạn 3: Duy trì tải đỉnh
          { duration: RAMP_TIME, target: 0 },                          // Giai đoạn 4: Hạ tải dần về 0
        ],
        gracefulRampDown: '5s',
      },
    },
    thresholds: {
      http_req_failed: ['rate<0.01'],              // Lỗi HTTP < 1%
      http_req_duration: ['p(95)<50', 'p(99)<100'], // 95% request < 50ms
      stock_success_rate: ['rate>0.99'],           // Tỉ lệ nghiệp vụ thành công > 99%
    },
  };
})();

// ============================================================================
// 3. HÀM THỰC THI (VU ITERATION)
// ============================================================================
export default function () {
  const url = `${BASE_URL}/api/v1/inventories/stock?ticket_type_ids=${TICKET_TYPE_ID}`;
  const params = {
    headers: {
      'Accept': 'application/json',
    },
    tags: { name: 'GetCurrentStock' },
  };

  const res = http.get(url, params);

  stockRequestsTotal.add(1);
  stockLatency.add(res.timings.duration);

  // Kiểm tra tính toàn vẹn của kết quả
  const isOk = check(res, {
    'status is 200': (r) => r.status === 200,
    'has data response': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body && body.data && body.data.ticket_type_id === TICKET_TYPE_ID;
      } catch (e) {
        return false;
      }
    },
  });

  stockSuccessRate.add(isOk);

  // Nếu muốn pacing / delay giữa các request của cùng 1 VU
  if (SLEEP_MS > 0) {
    sleep(SLEEP_MS / 1000);
  }
}

// ============================================================================
// 4. BÁO CÁO SAU KHI CHẠY XONG (CUSTOM SUMMARY REPORT)
// ============================================================================
export function handleSummary(data) {
  const reqMetric = data.metrics.http_reqs || {};
  const durMetric = data.metrics.http_req_duration || {};
  const failMetric = data.metrics.http_req_failed || {};

  const getVal = (metric, key) => (metric && metric.values && metric.values[key] !== undefined && metric.values[key] !== null ? metric.values[key].toFixed(2) : '0.00');

  const totalReqs = reqMetric.values && reqMetric.values.count !== undefined ? reqMetric.values.count : 0;
  const rps = reqMetric.values && reqMetric.values.rate !== undefined ? reqMetric.values.rate.toFixed(2) : '0.00';
  const p90 = getVal(durMetric, 'p(90)');
  const p95 = getVal(durMetric, 'p(95)');
  const p99 = getVal(durMetric, 'p(99)');
  const avg = getVal(durMetric, 'avg');
  const min = getVal(durMetric, 'min');
  const max = getVal(durMetric, 'max');
  const failRate = failMetric.values && failMetric.values.rate !== undefined ? (failMetric.values.rate * 100).toFixed(2) : '0.00';

  // 1. Text Summary in ra console
  const terminalReport = `
================================================================================
                    BÁO CÁO HIỆU NĂNG GET CURRENT STOCK (K6)
================================================================================
Target URL         : ${BASE_URL}/api/v1/inventories/stock?ticket_type_ids=${TICKET_TYPE_ID}
Chế độ tải        : ${TEST_MODE} (VUs: ${TARGET_VUS}, Duration: ${DURATION})
--------------------------------------------------------------------------------
Tổng số request    : ${totalReqs} requests
Throughput (RPS)   : ${rps} req/sec  <--- Số request xử lý được mỗi giây
Tỉ lệ thất bại     : ${failRate}%
--------------------------------------------------------------------------------
Độ trễ Response Time:
  - Trung bình (Avg) : ${avg} ms
  - Nhanh nhất (Min) : ${min} ms
  - Chậm nhất (Max)  : ${max} ms
  - Percentile 90    : ${p90} ms
  - Percentile 95    : ${p95} ms
  - Percentile 99    : ${p99} ms
================================================================================
`;

  // 2. HTML Report tự chứa (Standalone HTML dashboard)
  const htmlDashboard = `<!DOCTYPE html>
<html lang="vi">
<head>
  <meta charset="UTF-8">
  <title>K6 Performance Report - Inventory Stock API</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background: #0f172a; color: #f8fafc; padding: 30px; margin: 0; }
    .container { max-width: 900px; margin: 0 auto; background: #1e293b; border-radius: 12px; padding: 30px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
    h1 { color: #38bdf8; margin-top: 0; border-bottom: 2px solid #334155; padding-bottom: 12px; }
    .meta { color: #94a3b8; font-size: 14px; margin-bottom: 24px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 30px; }
    .card { background: #0f172a; border: 1px solid #334155; border-radius: 8px; padding: 18px; }
    .card .title { font-size: 13px; color: #94a3b8; text-transform: uppercase; margin-bottom: 8px; }
    .card .value { font-size: 28px; font-weight: bold; color: #38bdf8; }
    .card.success .value { color: #4ade80; }
    .card.warning .value { color: #facc15; }
    .card.danger .value { color: #f87171; }
    table { width: 100%; border-collapse: collapse; margin-top: 15px; }
    th, td { text-align: left; padding: 12px; border-bottom: 1px solid #334155; }
    th { color: #94a3b8; font-weight: 600; font-size: 14px; }
    td { font-size: 15px; }
    .badge { display: inline-block; padding: 3px 8px; border-radius: 4px; font-size: 12px; font-weight: 600; background: #334155; color: #38bdf8; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Báo Cáo Hiệu Năng API Get Current Stock</h1>
    <div class="meta">
      <div><b>Target URL:</b> <code>${BASE_URL}/api/v1/inventories/stock?ticket_type_ids=${TICKET_TYPE_ID}</code></div>
      <div><b>Kịch bản:</b> ${TEST_MODE} (Target VUs: <b>${TARGET_VUS}</b>, Duration: <b>${DURATION}</b>, Ramp: <b>${RAMP_TIME}</b>)</div>
      <div><b>Thời gian chạy:</b> ${new Date().toLocaleString()}</div>
    </div>

    <div class="grid">
      <div class="card success">
        <div class="title">Throughput (RPS)</div>
        <div class="value">${rps} <span style="font-size:16px;">req/s</span></div>
      </div>
      <div class="card">
        <div class="title">Tổng số Request</div>
        <div class="value">${totalReqs}</div>
      </div>
      <div class="card warning">
        <div class="title">P95 Response Time</div>
        <div class="value">${p95} <span style="font-size:16px;">ms</span></div>
      </div>
      <div class="card ${failRate > 0 ? 'danger' : 'success'}">
        <div class="title">Tỉ Lệ Lỗi (Error Rate)</div>
        <div class="value">${failRate}%</div>
      </div>
    </div>

    <h2>Chi Tiết Phân Phối Độ Trễ (Latency Distribution)</h2>
    <table>
      <thead>
        <tr>
          <th>Chỉ số (Metric)</th>
          <th>Thời gian (ms)</th>
          <th>Đánh giá với Redis Cache</th>
        </tr>
      </thead>
      <tbody>
        <tr><td>Nhanh nhất (Min)</td><td><b>${min} ms</b></td><td><span class="badge">Siêu tốc</span></td></tr>
        <tr><td>Trung bình (Avg)</td><td><b>${avg} ms</b></td><td><span class="badge">Rất tốt</span></td></tr>
        <tr><td>Percentile 90 (P90)</td><td><b>${p90} ms</b></td><td><span class="badge">90% request</span></td></tr>
        <tr><td>Percentile 95 (P95)</td><td><b>${p95} ms</b></td><td><span class="badge">95% request</span></td></tr>
        <tr><td>Percentile 99 (P99)</td><td><b>${p99} ms</b></td><td><span class="badge">99% request</span></td></tr>
        <tr><td>Chậm nhất (Max)</td><td><b>${max} ms</b></td><td><span class="badge">Bao gồm cache miss đầu tiên</span></td></tr>
      </tbody>
    </table>
  </div>
</body>
</html>`;

  return {
    'stdout': terminalReport,
    'k6/report.html': htmlDashboard,
    'k6/summary.json': JSON.stringify(data, null, 2),
  };
}
