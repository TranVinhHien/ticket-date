/**
 * init-data.js
 * ============
 * Script khởi tạo dữ liệu trước khi chạy k6 load test.
 * Chạy bằng Node.js (không phải k6): node k6/setup/init-data.js
 *
 * Thực hiện:
 *   1. Đăng ký + đăng nhập admin user (để tạo organizer, venue, event, inventory)
 *   2. Tạo organizer
 *   3. Tạo venue
 *   4. Tạo N event (mỗi event có 2 ticket_type)
 *   5. Tạo inventory cho từng ticket_type
 *   6. Tạo M test users và lấy token
 *   7. Lưu toàn bộ dữ liệu ra file JSON (k6/setup/test-data.json)
 *
 * Cấu hình qua biến môi trường:
 *   BASE_URL          - URL gốc của API (default: http://localhost:8000)
 *   EVENT_URL         - URL event service (default: BASE_URL)
 *   INVENTORY_URL     - URL inventory service (default: BASE_URL)
 *   ORDER_URL         - URL order service (default: BASE_URL)
 *   ADMIN_EMAIL       - Email admin (default: tự generate)
 *   ADMIN_PASSWORD    - Password admin (default: Admin@123456)
 *   NUM_EVENTS        - Số event cần tạo (default: 1)
 *   NUM_USERS         - Số test user cần tạo (default: 10)
 *   INVENTORY_QTY     - Số lượng vé mỗi ticket_type (default: 10000)
 */

const fs = require("fs");
const path = require("path");
const https = require("https");
const http = require("http");

// ─── Config ──────────────────────────────────────────────────────────────────
const BASE_URL = process.env.BASE_URL || "http://localhost:8000";
const EVENT_URL = process.env.EVENT_URL || BASE_URL;
const INVENTORY_URL = process.env.INVENTORY_URL || BASE_URL;
const ORDER_URL = process.env.ORDER_URL || BASE_URL;
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || "Admin@123456";
const NUM_EVENTS = parseInt(process.env.NUM_EVENTS || "4", 10);
const NUM_USERS = parseInt(process.env.NUM_USERS || "10", 10);
const INVENTORY_QTY = parseInt(process.env.INVENTORY_QTY || "1500", 10);

const OUTPUT_FILE = path.join(__dirname, "test-data.json");

// ─── Helpers ─────────────────────────────────────────────────────────────────
function uid() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 7);
}

/**
 * Minimal HTTP client dùng built-in Node.js http/https module.
 * Trả về Promise<{ status, body }> (body đã parse JSON nếu có thể).
 */
function request(method, url, body, headers = {}) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const isHttps = parsed.protocol === "https:";
    const lib = isHttps ? https : http;

    const payload = body ? JSON.stringify(body) : null;
    const reqHeaders = {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...headers,
    };
    if (payload) {
      reqHeaders["Content-Length"] = Buffer.byteLength(payload);
    }

    const options = {
      hostname: parsed.hostname,
      port: parsed.port || (isHttps ? 443 : 80),
      path: parsed.pathname + parsed.search,
      method: method.toUpperCase(),
      headers: reqHeaders,
    };

    const req = lib.request(options, (res) => {
      let data = "";
      res.on("data", (chunk) => (data += chunk));
      res.on("end", () => {
        let parsed_body;
        try {
          parsed_body = JSON.parse(data);
        } catch {
          parsed_body = data;
        }
        resolve({ status: res.statusCode, body: parsed_body });
      });
    });

    req.on("error", reject);
    if (payload) req.write(payload);
    req.end();
  });
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

function log(msg) {
  console.log(`[${new Date().toISOString()}] ${msg}`);
}

function logError(msg, detail) {
  console.error(`[ERROR] ${msg}`, detail ? JSON.stringify(detail, null, 2) : "");
}

// ─── API Wrappers (theo đúng Bruno) ──────────────────────────────────────────

/**
 * POST /api/v1/auth/register
 * Body: { email, password, full_name, avatar_url, phone_number, current_lat, current_lng, home_address }
 */
async function registerUser(email, password, fullName) {
  const res = await request("POST", `${BASE_URL}/api/v1/auth/register`, {
    email,
    password,
    full_name: fullName,
    avatar_url: "",
    phone_number: "",
    current_lat: 10.7769,
    current_lng: 106.7009,
    home_address: "214 Dinh Bo Linh, Ho Chi Minh",
  });

  if (res.status !== 200 && res.status !== 201) {
    // Nếu user đã tồn tại thì bỏ qua
    if (res.status === 409 || (res.body && res.body.code === "409")) {
      log(`  User ${email} đã tồn tại, tiếp tục login...`);
      return null;
    }
    logError(`Register failed for ${email}`, { status: res.status, body: res.body });
    throw new Error(`Register failed: ${res.status}`);
  }
  return res.body;
}

/**
 * POST /api/v1/auth/login
 * Body: { email, password }
 * Response: { code, data: { token, refresh_token } }
 */
async function loginUser(email, password) {
  const res = await request("POST", `${BASE_URL}/api/v1/auth/login`, {
    email,
    password,
  });

  if (res.status !== 200) {
    logError(`Login failed for ${email}`, { status: res.status, body: res.body });
    throw new Error(`Login failed: ${res.status}`);
  }

  const body = res.body;
  if (!body || body.code !== "200" || !body.data || !body.data.token) {
    logError(`Login response invalid for ${email}`, body);
    throw new Error("Login response invalid");
  }

  return { token: body.data.token, refresh_token: body.data.refresh_token };
}

/**
 * POST /api/v1/organizers
 * Headers: Authorization: Bearer <token>
 * Body: { organization_name, description, logo_url, tax_code, address, status }
 */
async function createOrganizer(token, suffix = "") {
  const res = await request(
    "POST",
    `${EVENT_URL}/api/v1/organizers`,
    {
      organization_name: `LiveNation Vietnam ${suffix}`,
      description: "Leading entertainment and concert promoter in Vietnam",
      logo_url: "https://example.com/logo.png",
      tax_code: `010998877${uid().slice(0, 4)}`,
      address: "123 Nguyen Hue, District 1, Ho Chi Minh City",
      status: "ACTIVE",
    },
    { Authorization: `Bearer ${token}` }
  );

  if (res.status !== 200 && res.status !== 201) {
    logError("Create organizer failed", { status: res.status, body: res.body });
    throw new Error(`Create organizer failed: ${res.status}`);
  }

  const id = res.body?.data?.id;
  if (!id) {
    logError("Organizer ID not found in response", res.body);
    throw new Error("Organizer ID not found");
  }
  return id;
}

/**
 * POST /api/v1/venues
 * Headers: Authorization: Bearer <token>
 * Body: { organizer_id, name, address, city, country, map_url, latitude, longitude, capacity, description, status }
 */
async function createVenue(token, organizerId, suffix = "") {
  const res = await request(
    "POST",
    `${EVENT_URL}/api/v1/venues`,
    {
      organizer_id: organizerId,
      name: `Saigon Exhibition and Convention Center ${suffix}`,
      address: "799 Nguyen Van Linh, Tan Phu, District 7",
      city: "Ho Chi Minh City",
      country: "Vietnam",
      map_url: "https://maps.google.com/?q=SECC",
      latitude: 10.7327,
      longitude: 106.7214,
      capacity: 20000,
      description: "Premier international exhibition & convention center",
      status: "ACTIVE",
    },
    { Authorization: `Bearer ${token}` }
  );

  if (res.status !== 200 && res.status !== 201) {
    logError("Create venue failed", { status: res.status, body: res.body });
    throw new Error(`Create venue failed: ${res.status}`);
  }

  const id = res.body?.data?.id;
  if (!id) {
    logError("Venue ID not found in response", res.body);
    throw new Error("Venue ID not found");
  }
  return id;
}

/**
 * POST /api/v1/events
 * Headers: Authorization: Bearer <token>
 * Body: { organizer_id, venue_id, name, description, poster_url, start_at, end_at, status, ticket_types[] }
 * Response: { data: { id, slug, ticket_types: [{ id, sku, name }] } }
 */
async function createEvent(token, organizerId, venueId, index = 0) {
  const eventSuffix = uid();
  const res = await request(
    "POST",
    `${EVENT_URL}/api/v1/events`,
    {
      organizer_id: organizerId,
      venue_id: venueId,
      name: `Summer Sonic Music Festival ${eventSuffix}`,
      description: "The largest outdoor international music festival in Vietnam",
      poster_url: "https://example.com/poster-summer-sonic.jpg",
      start_at: "2027-08-15T16:00:00Z",
      end_at: "2027-08-15T23:30:00Z",
      status: "PUBLISHED",
      ticket_types: [
        {
          sku: `VIP-${eventSuffix}-A`,
          name: "VIP Standing Zone A",
          description: "Front stage access and welcome drink",
          price: 2500000,
          currency: "VND",
          max_per_order: 4,
          sale_start_at: "2026-06-01T09:00:00Z",
          sale_end_at: "2027-08-14T23:59:59Z",
          attributes: { zone: "A", fast_track: true },
          status: "ACTIVE",
        },
        {
          sku: `GA-${eventSuffix}-B`,
          name: "General Admission",
          description: "Standard festival entry",
          price: 800000,
          currency: "VND",
          max_per_order: 6,
          sale_start_at: "2026-06-01T09:00:00Z",
          sale_end_at: "2027-08-14T23:59:59Z",
          attributes: { zone: "GA" },
          status: "ACTIVE",
        },
      ],
    },
    { Authorization: `Bearer ${token}` }
  );

  if (res.status !== 200 && res.status !== 201) {
    logError(`Create event #${index} failed`, { status: res.status, body: res.body });
    throw new Error(`Create event failed: ${res.status}`);
  }

  const data = res.body?.data;
  if (!data || !data.id) {
    logError("Event data not found in response", res.body);
    throw new Error("Event data not found");
  }

  return {
    event_id: data.id,
    event_slug: data.slug,
    ticket_types: (data.ticket_types || []).map((tt) => ({
      id: tt.id,
      sku: tt.sku,
      name: tt.name,
    })),
  };
}

/**
 * POST /api/v1/inventories
 * Headers: Authorization: Bearer <token>, X-User-ID: <adminUserId>
 * Body: { reference_type, note, items: [{ ticket_type_id, total_quantity }] }
 * Response: { data: [{ id, ticket_type_id }] }
 */
async function createInventory(token, adminUserId, ticketTypeIds, quantity) {
  const res = await request(
    "POST",
    `${INVENTORY_URL}/api/v1/inventories`,
    {
      reference_type: "INITIAL_IMPORT",
      note: "Initial inventory batch creation for load test",
      items: ticketTypeIds.map((id) => ({
        ticket_type_id: id,
        total_quantity: quantity,
      })),
    },
    {
      Authorization: `Bearer ${token}`,
      "X-User-ID": adminUserId,
    }
  );

  if (res.status !== 200 && res.status !== 201) {
    logError("Create inventory failed", { status: res.status, body: res.body });
    throw new Error(`Create inventory failed: ${res.status}`);
  }

  const data = res.body?.data;
  if (!data || !Array.isArray(data)) {
    logError("Inventory data not found in response", res.body);
    throw new Error("Inventory data not found");
  }

  return data.map((item) => ({ inventory_id: item.id, ticket_type_id: item.ticket_type_id }));
}

/**
 * GET /api/v1/users/me
 * Headers: Authorization: Bearer <token>
 * Dùng để lấy user ID sau khi login
 */
async function getMyProfile(token) {
  const res = await request("GET", `${BASE_URL}/api/v1/users/me`, null, {
    Authorization: `Bearer ${token}`,
  });

  if (res.status !== 200) {
    logError("Get profile failed", { status: res.status, body: res.body });
    throw new Error(`Get profile failed: ${res.status}`);
  }

  const data = res.body?.data;
  if (!data || !data.id) {
    logError("User profile data not found", res.body);
    throw new Error("User profile data not found");
  }
  return data;
}

// ─── Main ─────────────────────────────────────────────────────────────────────
async function main() {
  log("=== Bắt đầu khởi tạo dữ liệu cho k6 load test ===");
  log(`BASE_URL=${BASE_URL} | EVENT_URL=${EVENT_URL} | INVENTORY_URL=${INVENTORY_URL} | ORDER_URL=${ORDER_URL}`);
  log(`NUM_EVENTS=${NUM_EVENTS} | NUM_USERS=${NUM_USERS} | INVENTORY_QTY=${INVENTORY_QTY}`);

  const result = {
    base_url: BASE_URL,
    event_url: EVENT_URL,
    inventory_url: INVENTORY_URL,
    order_url: ORDER_URL,
    admin: null,
    organizer_id: null,
    venue_id: null,
    events: [],  // [{ event_id, event_slug, ticket_types: [{ id, name }], inventories }]
    users: [],   // [{ email, password, token, refresh_token, user_id }]
    created_at: new Date().toISOString(),
    config: { num_events: NUM_EVENTS, num_users: NUM_USERS, inventory_qty: INVENTORY_QTY },
  };

  // ── Step 1: Admin user ───────────────────────────────────────────────────
  log("\n[Step 1] Tạo/login admin user...");
  const adminEmail = process.env.ADMIN_EMAIL || `admin-k6-${uid()}@test.com`;
  const adminPassword = ADMIN_PASSWORD;

  await registerUser(adminEmail, adminPassword, "Admin K6 Test");
  const adminAuth = await loginUser(adminEmail, adminPassword);
  const adminProfile = await getMyProfile(adminAuth.token);

  result.admin = {
    email: adminEmail,
    password: adminPassword,
    user_id: adminProfile.id,
    token: adminAuth.token,
    refresh_token: adminAuth.refresh_token,
  };
  log(`  ✓ Admin user: ${adminEmail} (id=${adminProfile.id})`);

  // ── Step 2: Organizer ────────────────────────────────────────────────────
  log("\n[Step 2] Tạo organizer...");
  const organizerId = await createOrganizer(adminAuth.token, uid());
  result.organizer_id = organizerId;
  log(`  ✓ Organizer: ${organizerId}`);

  // ── Step 3: Venue ────────────────────────────────────────────────────────
  log("\n[Step 3] Tạo venue...");
  const venueId = await createVenue(adminAuth.token, organizerId, uid());
  result.venue_id = venueId;
  log(`  ✓ Venue: ${venueId}`);

  // ── Step 4: Events + Inventory ───────────────────────────────────────────
  log(`\n[Step 4] Tạo ${NUM_EVENTS} event(s) + inventory...`);

  for (let i = 0; i < NUM_EVENTS; i++) {
    log(`  Event #${i + 1}/${NUM_EVENTS}...`);

    const eventData = await createEvent(adminAuth.token, organizerId, venueId, i);
    log(`    ✓ Event: ${eventData.event_id} (slug=${eventData.event_slug})`);
    log(`    ✓ Ticket types: ${eventData.ticket_types.map((tt) => `${tt.id}(${tt.name})`).join(", ")}`);

    const ticketTypeIds = eventData.ticket_types.map((tt) => tt.id);
    const inventories = await createInventory(
      adminAuth.token,
      adminProfile.id,
      ticketTypeIds,
      INVENTORY_QTY
    );
    log(`    ✓ Inventories: ${inventories.map((inv) => inv.inventory_id).join(", ")}`);

    result.events.push({
      event_id: eventData.event_id,
      event_slug: eventData.event_slug,
      ticket_types: eventData.ticket_types,
      inventories,
    });

    // Rate limit buffer
    if (i < NUM_EVENTS - 1) await sleep(200);
  }

  // ── Step 5: Test users ───────────────────────────────────────────────────
  log(`\n[Step 5] Tạo ${NUM_USERS} test user(s)...`);
  const userPassword = "TestUser@123";
  const CONCURRENCY = 5; // Tạo tối đa 5 user song song

  for (let batch = 0; batch < NUM_USERS; batch += CONCURRENCY) {
    const batchEnd = Math.min(batch + CONCURRENCY, NUM_USERS);
    const promises = [];

    for (let i = batch; i < batchEnd; i++) {
      const email = `k6-user-${uid()}@test.com`;
      const fullName = `K6 Test User ${i + 1}`;
      promises.push(
        (async () => {
          await registerUser(email, userPassword, fullName);
          const auth = await loginUser(email, userPassword);
          const profile = await getMyProfile(auth.token);
          return {
            email,
            password: userPassword,
            user_id: profile.id,
            token: auth.token,
            refresh_token: auth.refresh_token,
          };
        })()
      );
    }

    const batchResults = await Promise.all(promises);
    result.users.push(...batchResults);
    log(`  ✓ Tạo ${batchResults.length} users (tổng: ${result.users.length}/${NUM_USERS})`);
    if (batchEnd < NUM_USERS) await sleep(300);
  }

  // ── Lưu kết quả ─────────────────────────────────────────────────────────
  log(`\n[Done] Lưu dữ liệu vào ${OUTPUT_FILE}...`);
  fs.writeFileSync(OUTPUT_FILE, JSON.stringify(result, null, 2), "utf8");

  log("\n=== Tóm tắt dữ liệu đã khởi tạo ===");
  log(`  Admin       : ${result.admin.email}`);
  log(`  Organizer   : ${result.organizer_id}`);
  log(`  Venue       : ${result.venue_id}`);
  log(`  Events      : ${result.events.length}`);
  result.events.forEach((e, i) => {
    log(`    [${i + 1}] event_id=${e.event_id}, slug=${e.event_slug}`);
    e.ticket_types.forEach((tt) => log(`        ticket_type: ${tt.id} (${tt.name})`));
    e.inventories.forEach((inv) => log(`        inventory  : ${inv.inventory_id}`));
  });
  log(`  Test users  : ${result.users.length}`);
  log(`  Output      : ${OUTPUT_FILE}`);
  log("\n✅ Khởi tạo dữ liệu hoàn tất! Chạy k6 test ngay bây giờ.");
}

main().catch((err) => {
  console.error("\n❌ Init data thất bại:", err.message);
  process.exit(1);
});
