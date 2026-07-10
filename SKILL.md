# PetroSync — SKILL.md
> Go API + HTMX Dashboard for fuel distribution tracking across RU II–VI.
> This file is the authoritative engineering reference for the `petrosync` repository.
> Read this before writing any Go, SQL, or template code.

---

## Table of Contents
1. [Project Identity](#1-project-identity)
2. [Locked Tech Stack](#2-locked-tech-stack)
3. [Repository Structure](#3-repository-structure)
4. [Database Layer Rules](#4-database-layer-rules)
5. [Go API Architecture](#5-go-api-architecture)
6. [Mobile API Contract](#6-mobile-api-contract)
7. [Authentication & Session Architecture](#7-authentication--session-architecture)
8. [RBAC Enforcement Rules](#8-rbac-enforcement-rules)
9. [Canonical State Machines](#9-canonical-state-machines)
10. [Business Logic Rules by Domain](#10-business-logic-rules-by-domain)
11. [Dashboard Architecture](#11-dashboard-architecture)
12. [Telegram Bot Rules](#12-telegram-bot-rules)
13. [Object Storage Rules — Garage](#13-object-storage-rules--garage)
14. [Valkey Architecture](#14-valkey-architecture)
15. [Real-time WebSocket Architecture](#15-real-time-websocket-architecture)
16. [Background Worker Architecture](#16-background-worker-architecture)
17. [Phase 1 — Core Loop](#17-phase-1--core-loop)
18. [Phase 2 — Safety Layer](#18-phase-2--safety-layer)
19. [Phase 3 — Intelligence](#19-phase-3--intelligence)
20. [Phase 4 — Enterprise](#20-phase-4--enterprise)
21. [Code Commenting Standards](#21-code-commenting-standards)
22. [Makefile Requirements](#22-makefile-requirements)
23. [Visual Design System](#23-visual-design-system)
24. [Forbidden Patterns](#24-forbidden-patterns)
25. [sqlc.yaml Reference](#25-sqlcyaml-reference)

---

## 1. Project Identity

| Field | Value |
|---|---|
| **Repository** | `petrosync` |
| **Companion repo** | `petrosync-android` — Kotlin Android app (separate repo) |
| **Domain** | Fuel distribution tracking — refinery to gas station |
| **Org model** | Multi-refinery, single-company (not SaaS, no tenant isolation) |
| **Refineries** | RU II Dumai, RU III Plaju, RU IV Cilacap, RU V Balikpapan, RU VI Balongan |
| **Fleet model** | Shared pool per island geography (no cross-sea assignment) |
| **Trip model** | Single-destination only — one DO, one truck, one station |
| **Compartments** | Multi-compartment per vehicle from day one |
| **Scan mechanism** | Static QR code at loading bays and station delivery points |
| **Volume source** | Weight bridge (primary); manual entry requires approval chain |
| **Photo storage** | Garage (S3-compatible, k3s deployed) |
| **Notifications** | Telegram bot — notify-only, no interactive commands |
| **Target deployment** | Single k3s cluster — ArgoCD + Woodpecker CI + Harbor |

---

## 2. Locked Tech Stack

### Backend API
| Component | Choice |
|---|---|
| Language | Go 1.25+ |
| HTTP framework | Gin |
| Database | PostgreSQL 16+ with PostGIS |
| DB access | sqlc v2 — no ORM |
| Migration | golang-migrate/migrate |
| Cache / Pub-sub | Valkey (`valkey-go` client — NOT go-redis) |
| Object storage | Garage via AWS SDK v2 (custom endpoint) |
| WebSocket | gorilla/websocket or nhooyr.io/websocket |
| Background jobs | robfig/cron v3 + goroutines |
| Telegram bot | go-telegram-bot-api/telegram-bot-api |
| Config | godotenv + envconfig |
| PDF generation | jung-kurt/gofpdf or chromedp |
| Decimal arithmetic | shopspring/decimal (volumes and weights — never float64) |

### Dashboard
| Component | Choice |
|---|---|
| Templating | Go `html/template` (server-rendered) |
| Interactivity | HTMX 2.x |
| CSS | Tailwind CSS v4 |
| CSS build | Bun (exclusive JS runtime for asset pipeline) |
| Map | Leaflet.js (JS island — not HTMX-managed) |
| Charts | Chart.js (reporting pages) |

### Infrastructure
| Component | Choice |
|---|---|
| Container runtime | k3s |
| GitOps | ArgoCD |
| CI | Woodpecker CI (path-based triggers) |
| Registry | Harbor |
| Ingress | Traefik |
| TLS | cert-manager |

---

## 3. Repository Structure

Go is at the repo root. No `backend/` or `server/` subdirectory — Go module roots not at repo root create import path issues.

```
petrosync/
├── cmd/
│   ├── api/
│   │   └── main.go              # HTTP server entrypoint
│   └── worker/
│       └── main.go              # Background worker entrypoint
├── internal/
│   ├── config/
│   │   └── config.go            # Env-based config struct (all fields explicit)
│   ├── db/                      # sqlc-generated — NEVER EDIT MANUALLY
│   │   ├── db.go
│   │   ├── models.go
│   │   ├── querier.go
│   │   └── *.sql.go
│   ├── handler/                 # Gin route handlers — thin, no business logic
│   │   ├── delivery_order.go
│   │   ├── driver.go
│   │   ├── station.go
│   │   ├── trip.go
│   │   ├── user.go
│   │   ├── vehicle.go
│   │   └── ws.go                # WebSocket upgrade handler
│   ├── middleware/
│   │   ├── auth.go              # JWT validation + session lookup
│   │   ├── rbac.go              # Role + scope enforcement
│   │   └── audit.go             # Writes to audit_log on state changes
│   ├── service/                 # All business logic lives here
│   │   ├── dispatch.go          # Candidate vehicle selection
│   │   ├── variance.go          # Variance engine + dispute trigger
│   │   ├── document.go          # PDF generation
│   │   ├── storage.go           # Garage upload/download + presigned URLs
│   │   ├── notification.go      # Telegram message dispatch
│   │   └── qr.go                # QR payload validation (bay + station)
│   ├── worker/                  # Background goroutine jobs
│   │   ├── escalation.go        # Weight bridge approval escalation
│   │   ├── deviation.go         # Route deviation alert
│   │   └── expiry.go            # SIM B2 + keur expiry notification
│   ├── bot/
│   │   └── telegram.go          # Notify-only Telegram bot
│   └── ws/
│       └── hub.go               # WebSocket hub + Valkey pub/sub bridge
├── templates/
│   ├── layout/
│   │   ├── base.html
│   │   └── sidebar.html
│   ├── pages/                   # One file per major page
│   └── partials/                # HTMX swap targets
├── static/
│   ├── css/
│   │   └── app.css              # Tailwind source (input file)
│   └── js/
│       └── map.js               # Leaflet map JS island
├── sql/
├── schema.sql                   # DDL source of truth (32 tables, 15 enums)
├── queries.sql                  # Full query reference (autogenerated)
│   └── queries/              # sqlc query source (33 files, 251 queries)
│       ├── 000001_extensions.up.sql
│       ├── 000001_extensions.down.sql
├── sqlc.yaml
├── Makefile
├── go.mod
├── go.sum
├── .woodpecker.yml
├── .env.example
└── .gitignore
```

### Woodpecker CI Path Triggers

```yaml
# .woodpecker.yml
steps:
  build-api:
    image: golang:1.23
    when:
      path:
        - "cmd/**"
        - "internal/**"
        - "migrations/**"
        - "schema.sql"
        - "sql/queries/**"
        - "go.mod"
        - "go.sum"

  build-css:
    image: oven/bun:latest
    when:
      path:
        - "static/css/**"
        - "templates/**"
```

---

## 4. Database Layer Rules

### Absolute Rules

1. **No ORM.** Never use GORM, Ent, or any ORM. `sqlc` exclusively.
2. **`schema.sql` is the DDL source of truth.** 32 domain tables, 36 GPS partitions, 15 enum types. Never define tables in Go.
3. **`sql/queries/` is the query source of truth.** 251 named queries. Never write raw SQL strings in Go files. Add to `sql/queries/` first, then `make sqlc`.
4. **Never edit `internal/db/` manually.** Entirely generated by sqlc. All changes start in `sql/queries/`.
5. **Migrations use golang-migrate/migrate.** Paired `.up.sql` / `.down.sql` files per schema change in `sql/migrations/`. Never modify merged migration files.
6. **Append-only tables — enforced at DB role level.** `petrosync_app` has no `UPDATE`/`DELETE` on: `trip_events`, `gps_events`, `audit_log`, `notification_log`. Do not attempt workarounds.
7. **GPS events are partitioned by month.** Always insert into parent `gps_events`. Never reference a specific partition directly in code.
8. **PostGIS geometry convention:** write with `ST_SetSRID(ST_MakePoint($lng, $lat), 4326)`, read with `ST_X(col)` as longitude and `ST_Y(col)` as latitude.
9. **All volume and weight values are `NUMERIC` → Go `decimal.Decimal`.** Never `float64` for fuel arithmetic. Use `shopspring/decimal` throughout the service layer.
10. **No FK on `gps_events.trip_id`** — partitioned table limitation in PostgreSQL. Validate trip existence in the service layer before inserting GPS events.

### Connection Configuration

```go
// pgx/v5 pool. Connection string from DATABASE_URL env var.
// Pool: min 5, max 25. Acquire timeout: 5s.
// Always propagate request context to every sqlc call.
// Never use context.Background() inside handlers.
```

### Transaction Rules

Wrap these in `pgx.Tx`:
- DO approval → reserve storage tank volume (atomic)
- Trip creation → mark DO `IN_PROGRESS` → mark vehicle `ASSIGNED` (atomic)
- Delivery confirmation → update station tank volume → update compartment delivery (atomic)

Never hold a transaction across an HTTP round-trip or external network call.

---

## 5. Go API Architecture

### URL Design

```
/api/v1/auth/login
/api/v1/auth/logout
/api/v1/auth/refresh
/api/v1/auth/change-password

/api/v1/refineries
/api/v1/refineries/:id/facilities
/api/v1/facilities/:id/storage-tanks
/api/v1/facilities/:id/loading-bays
/api/v1/facilities/:id/delivery-orders
/api/v1/facilities/:id/dispatch-candidates

/api/v1/delivery-orders
/api/v1/delivery-orders/:id
/api/v1/delivery-orders/:id/items
/api/v1/delivery-orders/:id/approve
/api/v1/delivery-orders/:id/assign
/api/v1/delivery-orders/:id/cancel

/api/v1/trips
/api/v1/trips/active
/api/v1/trips/:id
/api/v1/trips/:id/events          # append-only POST (mobile + server)
/api/v1/trips/:id/weight-bridge
/api/v1/trips/:id/seals
/api/v1/trips/:id/photos          # multipart POST from Android
/api/v1/trips/:id/documents
/api/v1/trips/:id/compartments

/api/v1/vehicles
/api/v1/vehicles/:id
/api/v1/vehicles/:id/compartments
/api/v1/vehicles/:id/maintenance

/api/v1/drivers
/api/v1/drivers/:id
/api/v1/drivers/:id/shift/start
/api/v1/drivers/:id/shift/end

/api/v1/stations
/api/v1/stations/:id
/api/v1/stations/:id/tanks
/api/v1/stations/:id/qr-codes

/api/v1/users
/api/v1/users/:id
/api/v1/users/:id/roles
/api/v1/users/:id/reset-password

/api/v1/reports/delivery-stats
/api/v1/reports/driver-compliance
/api/v1/reports/fleet-status

# High-frequency mobile endpoints
/api/v1/gps/batch                 # POST — GPS ping array from Android
/api/v1/qr/validate               # POST — QR payload validation before full event

/ws/trips/active                  # WebSocket — dashboard map feed
```

### Standard Response Envelope

```go
// Single resource
type Response struct {
    Data any `json:"data"`
}

// Collection
type ListResponse struct {
    Data any      `json:"data"`
    Meta PageMeta `json:"meta"`
}

type PageMeta struct {
    Page    int `json:"page"`
    PerPage int `json:"per_page"`
    Total   int `json:"total"`
}

// Error
type ErrorResponse struct {
    Error APIError `json:"error"`
}

type APIError struct {
    Code    string `json:"code"`    // SCREAMING_SNAKE_CASE
    Message string `json:"message"` // Human-readable English
}
```

### Handler Rules

- Handlers are thin. No SQL, no business logic.
- Responsibility: parse input → call service → write response.
- Bind JSON with `c.ShouldBindJSON`. Path params with `c.Param`. Query params with `c.ShouldBindQuery`.
- Return structured errors immediately. Never panic in a handler.
- Every handler that mutates state must write to `audit_log`.

### Error Codes

```
NOT_FOUND              resource does not exist
VALIDATION_ERROR       malformed request body or missing required field
UNAUTHORIZED           not authenticated
FORBIDDEN              authenticated but insufficient role or scope
CONFLICT               state machine violation
INSUFFICIENT_STOCK     storage tank volume check failed
VEHICLE_UNAVAILABLE    keur expired or status not AVAILABLE
DRIVER_UNAVAILABLE     SIM B2 expired or not on shift
QR_INVALID             payload unrecognised or wrong trip context
SEAL_MISMATCH          seal verification failed
VARIANCE_EXCEEDED      variance above configured tolerance
APPROVAL_REQUIRED      manual weight bridge reading pending approval
PHOTO_MISSING          mandatory photo not uploaded for this step
INTERNAL_ERROR         unexpected server error
```

---

## 6. Mobile API Contract

This section defines what `petrosync-android` expects from this API. Keep it stable — breaking changes here require coordinating with the Android repo.

### Authentication Header

All authenticated mobile requests must include:
```
Authorization: Bearer <access_token>
```
Access token is a JWT. Refresh via `POST /api/v1/auth/refresh`.

### GPS Batch Endpoint

```
POST /api/v1/gps/batch
Content-Type: application/json

[
  {
    "event_uuid":      "550e8400-e29b-41d4-a716-446655440000",
    "trip_id":         42,
    "latitude":        -7.1234567,
    "longitude":       108.9876543,
    "speed_kmh":       72.5,
    "heading_deg":     183.2,
    "accuracy_m":      8.0,
    "event_timestamp": "2026-06-26T10:30:00+07:00"
  }
]

Response 202 Accepted:
{ "data": { "accepted": 10, "duplicates": 0 } }
```

Server processes the batch in a single transaction. Duplicate `event_uuid` values are silently skipped and counted in `duplicates`. Never return an error for duplicates — the Android sync protocol relies on idempotency.

### Trip Event Endpoint

```
POST /api/v1/trips/:id/events
Content-Type: application/json

{
  "event_uuid":      "550e8400-...",
  "event_type":      "COMPARTMENT_SEALED",
  "event_timestamp": "2026-06-26T10:35:00+07:00",
  "location": {
    "latitude":  -7.7250,
    "longitude": 108.9916
  },
  "payload": {
    "compartment_id":    3,
    "seal_number":       "SL-2026-00142",
    "qr_payload":        "LB-FAC-CLP-BAY01-..."
  }
}

Response 200 OK:  { "data": { "event_id": 88, "duplicate": false } }
Response 200 OK:  { "data": { "event_id": 72, "duplicate": true  } }
```

Never return 4xx for a duplicate event UUID. Return 200 with `duplicate: true`.

### Photo Upload Endpoint

```
POST /api/v1/trips/:id/photos
Content-Type: multipart/form-data

Fields:
  event_type:     (string) WEIGHT_BRIDGE_TARE | WEIGHT_BRIDGE_GROSS |
                           COMPARTMENT_SEALED | STATION_TANK_BEFORE |
                           PUMP_METER_READING | STATION_TANK_AFTER |
                           VARIANCE_EVIDENCE
  compartment_id: (int, optional) — required for COMPARTMENT_SEALED
  taken_at:       (string ISO 8601) — from device EXIF
  photo:          (file) JPEG, max 5MB

Response 201 Created:
{ "data": { "photo_id": 14 } }
```

Server uploads to Garage and inserts into `trip_photos`. Photo is not stored in the API server — it is streamed directly to Garage.

### Active Trip Polling

Android polls for its assigned trip on app foreground:
```
GET /api/v1/trips/active?driver_id={id}

Response 200:
{
  "data": {
    "trip_id": 42,
    "status":  "IN_TRANSIT",
    "do_number": "DO-RU4-2026-00023",
    "destination": { "name": "SPBU Solo Laweyan", "station_id": 9 },
    "compartments": [
      { "id": 3, "number": 1, "fuel_type": "PERTALITE", "capacity_l": 12000 },
      { "id": 4, "number": 2, "fuel_type": "BIOSOLAR",  "capacity_l": 12000 }
    ]
  }
}

Response 204 No Content — no active trip for this driver
```

### QR Validation

Android pre-validates QR payloads before submitting the full trip event:
```
POST /api/v1/qr/validate
Content-Type: application/json

{ "trip_id": 42, "qr_payload": "LB-FAC-CLP-BAY01-...", "context": "LOADING_BAY" }

Response 200: { "data": { "valid": true,  "location_name": "Cilacap Bay 01" } }
Response 200: { "data": { "valid": false, "reason": "QR_INVALID" } }
```

`context` is either `LOADING_BAY` or `STATION`.

---

## 7. Authentication & Session Architecture

### Dashboard (HTMX)

- Session-based. Cookie + server-side session in Valkey.
- Session key: `sess:{uuid}` → JSON `{user_id, role_grants[], expires_at}`. TTL 8 hours.
- Cookie: `HttpOnly`, `Secure`, `SameSite=Strict`.
- `force_password_change = TRUE` → redirect all routes to `/change-password`.
- Login: `GetUserByUsername` → bcrypt compare → write Valkey session → set cookie.

### Android (JWT)

- Access token: JWT HS256, 30-minute TTL. Signed with `JWT_SECRET` env var.
- Refresh token: opaque 64-char hex, 30-day TTL, stored as `jwt:refresh:{token}` → `user_id` in Valkey.
- JWT payload: `{ sub: user_id, roles: [{role, scope_type, scope_id}], exp, iat }`.
- Android stores both tokens in Android Keystore — never SharedPreferences. (Enforced in petrosync-android repo.)
- Middleware caches user active status in Valkey for 5 minutes: `user:active:{user_id}`.
- `DRIVER` JWT is rejected for all dashboard routes — return `403 FORBIDDEN`.

### Password Reset Flow

1. Admin: dashboard → user profile → Reset Password.
2. Service generates 12-char random alphanumeric temporary password.
3. `UpdateUserPassword` (bcrypt) + `SetForcePasswordChange(TRUE)`.
4. Send Telegram DM to user's `telegram_user_id` with temp password.
5. If no Telegram linked: show temp password in admin dashboard UI (one-time display, not stored in plaintext).
6. User logs in → forced password change screen.

### Telegram Bot Linking

1. Admin creates user → `CreateTelegramLinkToken` (48-hour TTL).
2. Admin manually shares token with the user via Telegram.
3. User sends `/link <token>` to bot.
4. Bot: `GetValidTelegramLinkToken` → `LinkTelegramAccount` + `UseTelegramLinkToken`.
5. Nightly cron: `DeleteExpiredTelegramLinkTokens`.

---

## 8. RBAC Enforcement Rules

### Role Hierarchy

```
SYSTEM_ADMIN      COMPANY scope   — all access
REFINERY_ADMIN    REFINERY scope  — escalation approver
FACILITY_MANAGER  FACILITY scope  — first approver for manual weight bridge
FACILITY_OPERATOR FACILITY scope  — DO management, weight bridge entry
DEPOT_STAFF       DEPOT scope     — vehicle maintenance, driver shifts
STATION_MANAGER   STATION scope   — raises DOs, confirms deliveries
DRIVER            COMPANY scope   — Android app only, no dashboard
```

### Middleware Chain

```
Request → JWTMiddleware/SessionMiddleware → RBACMiddleware(role, scopeType, scopeResolver) → Handler
```

`scopeResolver` extracts scope ID from the request path params.

### Enforcement Rules

- `SYSTEM_ADMIN` bypasses all scope checks.
- `REFINERY_ADMIN` has implicit read access to all facilities within their refinery.
- `DRIVER` JWT returns `403` on any dashboard route.
- Station operators see only their scoped station's data — all list queries must include `station_id` from `user_role_grants`.
- Role cache key: `rbac:{user_id}` → serialised `[]RoleGrant`, TTL 5 minutes. Revocations take effect within 5 minutes.

---

## 9. Canonical State Machines

Locked. Never add, remove, or rename states without updating the DB enum in `schema.sql` first.

### Delivery Order

```
DRAFT → PENDING_APPROVAL → APPROVED → ASSIGNED → IN_PROGRESS
  → DELIVERED → RECONCILED → CLOSED
  → DISPUTED (variance > tolerance or seal mismatch)
  → CLOSED (after supervisor resolution)

CANCELLED ← any state before IN_PROGRESS
```

| Transition | Actor | Guard |
|---|---|---|
| DRAFT → PENDING_APPROVAL | FACILITY_OPERATOR | At least one DO item exists |
| PENDING_APPROVAL → APPROVED | FACILITY_MANAGER+ | — |
| PENDING_APPROVAL → DRAFT | FACILITY_MANAGER+ | Rejection with note |
| APPROVED → ASSIGNED | FACILITY_OPERATOR | Vehicle + driver selected |
| ASSIGNED → IN_PROGRESS | system | On trip LOADING event |
| IN_PROGRESS → DELIVERED | system | On trip DELIVERED status |
| DELIVERED → RECONCILED | system | All compartments within tolerance |
| DELIVERED → DISPUTED | system | Any compartment exceeds tolerance |
| DISPUTED → CLOSED | REFINERY_ADMIN | Manual override with note |
| RECONCILED → CLOSED | system | Auto-close after reconciliation |
| Any pre-IN_PROGRESS → CANCELLED | FACILITY_OPERATOR+ | — |

### Trip

```
CREATED → DRIVER_ACKNOWLEDGED → PRE_TRIP_INSPECTION → LOADING → LOADED
  → IN_TRANSIT → ARRIVED → UNLOADING → DELIVERED → RECONCILED → CLOSED
  → DISPUTED
CANCELLED → (auto-creates RETURN_TO_FACILITY trip)
```

| Transition | Trigger | Hard Requirements |
|---|---|---|
| LOADING → LOADED | Loading bay QR scan | Tare + gross readings present; mandatory photos (TARE, GROSS, COMPARTMENT_SEALED per compartment) uploaded; if MANUAL method: approval status = APPROVED |
| ARRIVED → UNLOADING | Station QR scan | QR payload matches trip destination station |
| UNLOADING → DELIVERED | Driver confirms | STATION_TANK_BEFORE, PUMP_METER_READING, STATION_TANK_AFTER photos present; all seals verified (none MISMATCHED/BROKEN/MISSING) |
| DELIVERED → RECONCILED | system variance engine | All compartments within `variance_tolerance_pct` |
| DELIVERED → DISPUTED | system variance engine | Any compartment outside tolerance |
| CANCELLED | FACILITY_OPERATOR+ | If cancelled during or after LOADING: auto-creates RETURN_TO_FACILITY child trip. Cancellation before LOADING does not create a return trip. |

---

## 10. Business Logic Rules by Domain

### Weight Bridge Approval Chain

- `WEIGHT_BRIDGE` method → auto-approved. No user action.
- `MANUAL_APPROVED` method → enters `PENDING`.
- Worker runs every 30 minutes. Escalation window from `system_settings.approval_escalation_hours` (default: `2`).
- If `created_at + escalation_hours < NOW()` and still `PENDING` → `EscalateWeightBridgeReading` → Telegram DM to `REFINERY_ADMIN`.
- Trip cannot transition `LOADING → LOADED` while any weight bridge reading is `PENDING` or `ESCALATED`.

### Variance Engine

- Runs after all compartments in a trip reach `DELIVERED` state.
- Per compartment: `variance_pct = |loaded_vol - delivered_vol| / loaded_vol × 100`.
- Tolerance from `system_settings.variance_tolerance_pct` (default: `0.3`).
- Any compartment `variance_pct > tolerance` → compartment → `DISPUTED` → trip → `DISPUTED` → DO → `DISPUTED` → Telegram `VARIANCE_FLAGGED`.
- Temperature correction if `ambient_temp_celsius` recorded: `corrected_vol = net_kg / (density_at_15c × (1 + 0.00065 × (temp - 15)))`.
- All calculations: `decimal.Decimal`. Never `float64`.

### Dispatch Candidate Selection

- Call `ListDispatchCandidateVehicles(facilityID, limit)`.
- Post-filter: vehicle must have at least one compartment per fuel type in the DO items.
- Block if `sim_b2_expiry < CURRENT_DATE` or `keur_expiry < CURRENT_DATE`.
- Show top 5 to dispatcher. No auto-assignment without human confirmation in Phase 1.

### Storage Tank Reservation

- DO approval → `ReserveStorageTankVolume` for each item. Fail if insufficient.
- Trip LOADED → `DeductStorageTankVolume` (decrements + clears reservation).
- DO cancelled → `ReleaseStorageTankReservation`.
- Return trip delivery → `CreditStorageTankVolume`.

### Mandatory Photo Enforcement

**Before LOADING → LOADED:**
- `WEIGHT_BRIDGE_TARE` photo present.
- `WEIGHT_BRIDGE_GROSS` photo present.
- `COMPARTMENT_SEALED` photo present for every compartment.

**Before UNLOADING → DELIVERED:**
- `STATION_TANK_BEFORE` photo present.
- `PUMP_METER_READING` photo present.
- `STATION_TANK_AFTER` photo present.

Use `GetMandatoryPhotoCheckByTrip`. Return `PHOTO_MISSING` if any required photo absent.

### QR Code Validation

Loading bay scan:
1. `GetLoadingBayByQRPayload(payload)`.
2. Verify `facility_id = trip.origin_facility_id`.
3. Verify trip status is `DRIVER_ACKNOWLEDGED` or `PRE_TRIP_INSPECTION`.
4. Insert `ARRIVED_AT_FACILITY` trip event.

Station delivery scan:
1. `GetStationByQRPayload(payload)`.
2. Verify `station_id = trip.destination_station_id`.
3. Verify trip status is `ARRIVED`.
4. Insert `UNLOADING_STARTED` trip event.

### Seal Tracking

- `POST /api/v1/trips/:id/seals` with `compartment_id` + `seal_number_issued` after loading.
- At station: driver submits `seal_number_verified`. Service calls `VerifySeal` — auto-sets `INTACT` or `MISMATCHED`.
- Any `MISMATCHED`: insert `SEAL_MISMATCH_FLAGGED` event + Telegram `SEAL_MISMATCH` + block transition to `DELIVERED`.

### Route Deviation Policy

> ⚠️ **Phase 2 feature.** Route deviation monitoring and escalation is not part of Phase 1. Implement in Phase 2 (§18).

Worker checks active trips every 60 seconds:
- `occurrence_count = 1` → log only (`CreateDeviationEvent`).
- `occurrence_count = 2` → log + dashboard alert badge.
- Sustained `>= route_deviation_alert_minutes` (default 15) → Telegram escalation to facility supervisor + `MarkDeviationTelegramNotified`.

### Audit Log

Write `InsertAuditLog` on: DO status changes, trip status changes, weight bridge create/approve/reject, user role changes, password resets, manual tank corrections. Capture `before_state`, `after_state` as JSONB, `ip_address` from request. Non-blocking — use goroutine channel.

---

## 11. Dashboard Architecture

### Template Rules

- All pages extend `templates/layout/base.html` using Go template `block`.
- HTMX swap targets live in `templates/partials/`.
- No business logic in templates. Templates only render data from handlers.
- Use `hx-target`, `hx-swap`, `hx-push-url` explicitly — never rely on HTMX defaults.
- Destructive actions: `hx-confirm`. Loading states: `htmx:beforeRequest`/`htmx:afterRequest` spinner partial.

### Map (JS Island)

- `<div id="map">` in `templates/pages/dashboard.html`.
- `static/js/map.js` initialises Leaflet, connects to `/ws/trips/active`.
- One Leaflet marker per `trip_id`. Updates move the marker. Removal only on trip close/cancel.
- HTMX must never manage `#map` or anything inside it.

### HTMX Patterns

```html
<!-- Table auto-refresh every 30s -->
<tbody hx-get="/partials/active-trips"
       hx-trigger="every 30s"
       hx-swap="innerHTML">

<!-- DO approval -->
<button hx-post="/api/v1/delivery-orders/{{.ID}}/approve"
        hx-target="#do-{{.ID}}-row"
        hx-swap="outerHTML"
        hx-confirm="Approve this delivery order?">
  Approve
</button>
```

### Page Map

| URL | Purpose | Min Role |
|---|---|---|
| `/` | Company-wide ops overview + live map | REFINERY_ADMIN |
| `/facilities/:id` | Facility dashboard | FACILITY_OPERATOR |
| `/delivery-orders` | DO queue | FACILITY_OPERATOR |
| `/delivery-orders/:id` | DO detail + approval + assign | FACILITY_MANAGER |
| `/trips` | Trip list | FACILITY_OPERATOR |
| `/trips/:id` | Trip detail: timeline, weight bridge, seals, photos | FACILITY_OPERATOR |
| `/trips/:id/weight-bridge` | Weight bridge entry | FACILITY_OPERATOR |
| `/fleet` | Vehicle list | FACILITY_OPERATOR |
| `/fleet/:id` | Vehicle detail + maintenance | DEPOT_STAFF |
| `/stations` | Station list + tank levels | STATION_MANAGER |
| `/stations/:id` | Station tanks + delivery history | STATION_MANAGER |
| `/reports` | Stats, compliance, fleet | REFINERY_ADMIN |
| `/users` | User management | SYSTEM_ADMIN |
| `/settings` | System settings | SYSTEM_ADMIN |

**Role-based redirect:** Users who log in and access `/` without sufficient role (below `REFINERY_ADMIN`) are redirected to their scoped landing page: `FACILITY_OPERATOR` → `/facilities/:id`, `STATION_MANAGER` → `/stations/:id`, `DEPOT_STAFF` → `/fleet`.

---

## 12. Telegram Bot Rules

- **Notify-only except account linking.** The only slash command is `/link` (for Telegram account linking, see §7). No other slash commands, no interactive flows beyond linking, no conversation state.
- Targets: facility group chats (one per facility, chat ID in `system_settings`) + individual DMs.
- All sent messages recorded in `notification_log` via `InsertNotification`.
- On send failure: log error, set `delivery_status = FAILED`. No automatic retry.
- Bot token from `TELEGRAM_BOT_TOKEN` env var.

### Notification Trigger Map

| Event | Target | Template |
|---|---|---|
| DO raised | Facility group | 📋 DO {do_number} raised — {fuel} {volume}L → {station} |
| DO approved | Facility group | ✅ DO {do_number} approved |
| Trip assigned | Driver DM | 🚛 Trip assigned: {do_number} → {station} |
| Loading complete | Station Manager DM | 🚛 Truck {plate} loaded, en route |
| Trip departed | Facility group | 🚛 Truck {plate} departed → {station} |
| Delivery confirmed | Facility group + Station DM | ✅ {volume}L delivered to {station}. Variance: {pct}% |
| Variance flagged | Supervisor DM + group | ⚠️ Variance: Trip {id} compartment {n}: {pct}% |
| Seal mismatch | Supervisor DM | 🔒 Seal mismatch: Trip {id} compartment {n} |
| Route deviation escalation | Supervisor DM | 📍 Truck {plate} off-route for {min} min |
| Manual WBR pending | Facility Manager DM | ⚖️ Manual weight pending: Trip {id} |
| Manual WBR escalated | Refinery Admin DM | ⚠️ Weight bridge escalated: Trip {id} — no action in {h}h |
| SIM B2 expiring | Driver DM | ⏰ SIM B2 expires {date} |
| Keur expiring | Depot Staff DM | ⏰ Truck {plate} keur expires {date} |
| Return trip created | Facility group + Supervisor DM | ↩️ Return trip created for truck {plate} |

---

## 13. Object Storage Rules — Garage

- AWS SDK v2 with custom endpoint from `GARAGE_ENDPOINT` env var.
- Bucket: `petrosync`. Single bucket, key-namespaced.
- Object key convention:
  ```
  trips/{trip_id}/photos/{event_type}/{uuid}.jpg
  trips/{trip_id}/documents/{document_type}.pdf
  ```
- Never store file binary in PostgreSQL — only `garage_object_key`.
- Photo reads: presigned GET URL, 15-minute TTL. Never expose raw Garage endpoint publicly.
- Photo uploads from Android come through the API server (multipart POST). Android never calls Garage directly.

---

## 14. Valkey Architecture

Client: `valkey-go`. Single pool, 10 connections max.

### Key Namespaces

```
sess:{uuid}                   → dashboard session JSON, TTL 8h
jwt:refresh:{token}           → user_id string, TTL 30d
rbac:{user_id}                → []RoleGrant JSON, TTL 5m
user:active:{user_id}         → "1" or "0", TTL 5m (avoids DB hit per request)
trip:active                   → SET of active trip IDs
ws:trip:{trip_id}             → pub/sub channel for WebSocket hub
deviation:count:{trip_id}     → deviation occurrence count, TTL 24h
```

### Pub/Sub for Real-time Map

GPS batch ingestion → `PUBLISH ws:trip:{trip_id} <json>` → WebSocket hub goroutine → fan out to all connected dashboard clients.

---

## 15. Real-time WebSocket Architecture

```
Android GPS batch POST
  → Insert gps_events (batch tx)
  → PUBLISH ws:trip:{trip_id} {lat, lng, speed, plate, status}
      → WebSocket Hub (goroutine, subscribed to ws:trip:*)
          → Fan-out to all dashboard clients watching that trip
              → Leaflet marker update
```

Hub maintains `map[tripID][]*websocket.Conn`. Ping interval: 30s. Read deadline: 60s.

WebSocket message payload:
```json
{
  "trip_id":      42,
  "plate_number": "B 1234 ABC",
  "driver_name":  "Budi Santoso",
  "lat":          -7.1234567,
  "lng":          108.9876543,
  "speed_kmh":    72.5,
  "status":       "IN_TRANSIT",
  "destination":  "SPBU Solo Laweyan",
  "last_gps_at":  "2026-06-26T10:30:00Z"
}
```

---

## 16. Background Worker Architecture

`cmd/worker/main.go` runs all background jobs with `robfig/cron v3`.

```go
cron.New(cron.WithLocation(time.LoadLocation("Asia/Jakarta")))
```

| Job | Schedule (WIB) | Function |
|---|---|---|
| Weight bridge escalation | Every 30 min | `worker.CheckWeightBridgeEscalations()` |
| Route deviation alert | Every 60 sec | `worker.CheckRouteDeviations()` |
| SIM B2 expiry notification | Daily 07:00 | `worker.NotifyExpiringLicenses()` |
| Keur expiry notification | Daily 07:00 | `worker.NotifyExpiringKeur()` |
| Telegram token cleanup | Nightly 02:00 | `worker.CleanupExpiredLinkTokens()` |
| GPS partition pre-creation | 1st of month 01:00 | `worker.EnsureNextMonthGPSPartition()` |

Each job runs in its own goroutine. Use `recover()` in every job — a panic must never crash the worker process. Log start, completion, and duration of every job.

---

## 17. Phase 1 — Core Loop

**Goal:** Full end-to-end trip — DO creation → loading → transport → delivery → reconciliation. Manual DO only, single facility, no document generation, no geofencing.

### API Tasks

- [x] Project scaffold: `go.mod`, package layout, Gin router, config struct, `.env.example`
- [x] golang-migrate migrations: 15 paired files in `sql/migrations/` (from `schema.sql`)
- [x] sqlc generation: `sqlc.yaml` → `make sqlc` → verify `internal/db/`
- [x] Valkey client + session middleware (dashboard)
- [x] JWT middleware (Android) + refresh endpoint
- [x] RBAC middleware with scope resolver
- [x] Auth handlers: login, logout, change-password, refresh
- [ ] Telegram bot linking: token create endpoint + bot `/link` handler
- [x] Regions + fuel types (read-only)
- [x] Refinery + facility endpoints (read-only Phase 1)
- [x] Storage tank endpoints: list, available volume
- [x] Vehicle CRUD + status update + location update
- [x] Vehicle compartment CRUD
- [x] Driver CRUD + shift start/end
- [x] Gas station CRUD
- [x] Station tank list + dip reading update
- [x] Delivery order CRUD + approve + assign + cancel
- [x] DO items CRUD
- [x] Weight bridge: create, approve, escalate, reject
- [ ] Weight bridge approval chain service (block LOADED until approved)
- [x] Trip CRUD + status transitions (all state machine steps)
- [x] Trip event: POST with UUID idempotency check
- [x] QR validation service + `/api/v1/qr/validate` endpoint
- [x] Compartment delivery: create, update loaded/delivered volume, status
- [x] Compartment seal: issue, verify
- [ ] Storage tank reservation/deduction/credit (atomic transactions)
- [ ] Variance engine service
- [ ] Station tank volume update post-delivery
- [ ] Mandatory photo check service
- [ ] Photo upload endpoint → Garage client
- [x] GPS batch endpoint + Valkey pub/sub publish
- [x] WebSocket hub + `/ws/trips/active`
- [ ] Telegram notification service (all trigger types from Section 12)
- [ ] Audit log writes on all state changes
- [ ] User CRUD + role grant/revoke
- [ ] Password reset flow

### Dashboard Tasks

- [ ] Bun + Tailwind CSS setup → `make css`
- [ ] Base layout: sidebar, topbar, breadcrumb
- [ ] Login page + forced password change page
- [ ] Dashboard home: active trips, available vehicles, pending DOs
- [ ] Live map page: Leaflet JS island + WebSocket connection
- [ ] DO list + queue (HTMX table refresh)
- [ ] DO detail: approval form, vehicle/driver assignment
- [ ] Trip detail: event timeline, weight bridge, seals, photo gallery
- [ ] Weight bridge entry form
- [ ] Weight bridge approval queue page (FACILITY_MANAGER)
- [ ] Fleet list + vehicle detail + maintenance records
- [ ] Station list + tank level gauges
- [ ] Station detail + tank history
- [ ] User management: list, create, role assignment, password reset

---

## 18. Phase 2 — Safety Layer

**Goal:** Geofencing, route monitoring, document generation, seal enforcement, return-to-facility flow.

- [ ] Route deviation worker (active trip GPS analysis + deviation event creation)
- [ ] Route deviation escalation (Telegram after sustained threshold)
- [ ] Geofence auto-detection for `ARRIVED_AT_FACILITY` / `ARRIVED_AT_DESTINATION` events
- [ ] Return-to-facility trip auto-creation on `CANCELLED` mid-route
- [ ] Seal mismatch hard block on `DELIVERED` transition
- [ ] PDF generation service: Delivery Order, Bill of Lading, Delivery Receipt
- [ ] Document endpoints + presigned URL download
- [ ] Maintenance record CRUD + `UNDER_MAINTENANCE` dispatch block
- [ ] SIM B2 + keur expiry notification workers (activated in background worker)
- [ ] Deviation alert badge in HTMX dashboard partials

---

## 19. Phase 3 — Intelligence

**Goal:** Auto-DO, smart dispatch, forecasting, compliance scoring.

- [ ] Auto-DO cron: `ListStationTanksBelowReorderThreshold` → create DRAFT DOs
- [ ] Smart dispatch scoring: extend candidate query with variance history weight
- [ ] Consumption forecasting: linear regression on tank depletion → days of stock estimate
- [ ] Driver compliance score surfaced in reporting dashboard
- [ ] GPS partition pre-creation worker
- [ ] Monthly delivery stats report page
- [ ] Fleet utilisation report page
- [ ] Company-wide ops dashboard (cross-RU summary)
- [ ] Station operating hours warning (not block) on DO scheduling

---

## 20. Phase 4 — Enterprise

**Goal:** Regulatory output, inter-refinery transfers, ERP hooks.

- [ ] Inter-refinery transfer trip (`destination_type = REFINERY_FACILITY`)
- [ ] SPBU license number on delivery receipt PDFs
- [ ] BPH Migas report format export (format per regulator spec)
- [ ] ERP integration webhook (emit DO + delivery events to configured endpoint)
- [ ] Cross-RU consolidated report
- [ ] Audit log CSV export (date range)
- [ ] PostgreSQL analytics role (`petrosync_readonly`) for Metabase

---

## 21. Code Commenting Standards

### File-Level Doc Block

Every `.go` file starts with a package comment:

```go
// Package service implements the business logic layer for PetroSync.
// No SQL, no HTTP concerns belong here — only domain rules and orchestration.
package service
```

### Function-Level Doc

```go
// ApproveDO transitions a delivery order from PENDING_APPROVAL to APPROVED
// and atomically reserves the required fuel volume in the origin storage tanks.
//
// Errors:
//   - ErrNotFound if the delivery order does not exist.
//   - ErrConflict if the DO is not in PENDING_APPROVAL state.
//   - ErrInsufficientStock if any item volume cannot be reserved.
//   - ErrForbidden if the caller lacks FACILITY_MANAGER scope for the origin facility.
func (s *DeliveryOrderService) ApproveDO(ctx context.Context, doID, approverID int64) (*db.DeliveryOrder, error) {
```

### Inline Comments — Why, Not What

```go
// BAD:
counter++ // increment counter

// GOOD:
// Weight bridge tare reading is taken before loading, not at dispatch time.
// Using vehicle.tare_weight_kg as a fallback risks stale data if the truck
// was recently recalibrated — always require a fresh tare per trip.
tare, err := q.GetTareReadingByTrip(ctx, tripID)
```

### Test Functions

```go
// TestVarianceEngine_TemperatureCorrection verifies that high ambient
// temperatures produce corrected volumes below raw meter readings,
// consistent with the petroleum thermal expansion coefficient of 0.00065/°C.
func TestVarianceEngine_TemperatureCorrection(t *testing.T) {
```

---

## 22. Makefile Requirements

```makefile
.PHONY: run build build-worker sqlc migrate migrate-down migrate-create \
        test test-cover lint css css-watch seed docker-build docker-push

run:
	@which air > /dev/null 2>&1 && air || go run ./cmd/api

build:
	go build -o bin/petrosync-api ./cmd/api

build-worker:
	go build -o bin/petrosync-worker ./cmd/worker

sqlc:
	sqlc generate

migrate:
	migrate -path sql/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path sql/migrations -database "$$DATABASE_URL" down 1

migrate-create:
	migrate create -ext sql -dir sql/migrations -seq $(name)

test:
	go test ./... -v -race

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

css:
	bun run tailwindcss -i static/css/app.css -o static/css/dist.css --minify

css-watch:
	bun run tailwindcss -i static/css/app.css -o static/css/dist.css --watch

seed:
	psql "$$DATABASE_URL" -f sql/migrations/000015_seed.up.sql

docker-build:
	docker build -t petrosync-api:$(shell git rev-parse --short HEAD) .

docker-push:
	docker tag petrosync-api:$(shell git rev-parse --short HEAD) \
	    harbor.adevshankar.id/petrosync/api:$(shell git rev-parse --short HEAD)
	docker push harbor.adevshankar.id/petrosync/api:$(shell git rev-parse --short HEAD)
```

---

## 23. Visual Design System

Warm parchment + rust accents. Petroleum/industrial theme.

### Colour Palette

```css
:root {
  --color-bg:            #F5F0E8;
  --color-surface:       #FFFFFF;
  --color-surface-alt:   #EDE8DE;
  --color-border:        #D5CBBA;

  --color-primary:       #B5442A;
  --color-primary-dark:  #8C3320;
  --color-primary-light: #D4614B;

  --color-accent:        #1E3A5F;
  --color-accent-light:  #2A5080;

  --color-text-primary:  #1A1208;
  --color-text-muted:    #5C4A3A;
  --color-text-on-primary: #FFFFFF;

  --color-success:  #2E7D32;
  --color-warning:  #D97706;
  --color-danger:   #C62828;
  --color-info:     #1565C0;
}
```

### Status Colours

| Status | Colour |
|---|---|
| DRAFT, CREATED | neutral / text-muted |
| PENDING_APPROVAL, PENDING | warning — amber |
| APPROVED, ACKNOWLEDGED | info — navy |
| ASSIGNED, LOADING, LOADED | accent — petroleum blue |
| IN_TRANSIT, ARRIVED | primary — rust (active) |
| UNLOADING, DELIVERED | success — green |
| RECONCILED, CLOSED | text-muted — grey |
| DISPUTED, MISMATCHED | danger — red |
| CANCELLED | strikethrough + muted |
| ESCALATED | danger + pulsing dot |

### Component Rules

- Buttons: `rounded` (not `rounded-full`).
- Destructive actions: `bg-danger` + `hx-confirm`.
- Tables: zebra stripe with `bg-surface-alt` on even rows.
- Cards: `bg-surface rounded border border-border shadow-sm p-4`.
- Sidebar active: `bg-primary text-on-primary`.
- Monospace for: plate numbers, DO numbers, seal numbers, UUIDs.

---

## 24. Forbidden Patterns

| Pattern | Why |
|---|---|
| Raw SQL strings in `.go` files | All SQL lives in `sql/queries/` |
| `float64` for volume or weight | Use `decimal.Decimal` — fuel arithmetic requires exact decimals |
| `context.Background()` in handlers | Propagate request context always |
| `UPDATE`/`DELETE` on `trip_events`, `gps_events`, `audit_log`, `notification_log` | Append-only by design and DB role constraint |
| Referencing specific GPS partition tables | Always use parent `gps_events` |
| List queries without RBAC scope filter | A FACILITY_OPERATOR must never see another facility's data |
| Auto-approving `MANUAL_APPROVED` weight bridge readings | Human action required — FACILITY_MANAGER or REFINERY_ADMIN |
| Trip state transition that skips a step | Enforce the state machine table in Section 9 strictly |
| Storing photo binary in PostgreSQL | Garage only. Store `garage_object_key` |
| Synchronous GPS event processing on main thread | GPS batch: insert, publish to Valkey, return 202. Fire and accept. |
| `go-redis` client | Use `valkey-go` — licence difference is intentional |
| Any ORM dependency | `sqlc` only. `gorm`, `ent`, `bun ORM` are rejected |
| Trusting Android `event_timestamp` ordering by arrival | Process events ordered by `event_timestamp`, not `received_at` |
| Sending temp passwords to group chats | Temp passwords go to individual DMs only |
| Allowing DELIVERED transition without mandatory photos | `GetMandatoryPhotoCheckByTrip` must pass. No bypass. |
| Cross-island dispatch assignment | `max_assignment_radius_km` is enforced. Never override at service layer. |

---

## 25. sqlc.yaml Reference

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "schema.sql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
        emit_enum_valid_method: true
        emit_all_enum_values: true
        overrides:
          - column: "trip_events.payload"
            go_type: "encoding/json.RawMessage"
          - column: "audit_log.before_state"
            go_type: "encoding/json.RawMessage"
          - column: "audit_log.after_state"
            go_type: "encoding/json.RawMessage"
          - db_type: "numeric"
            go_type: "github.com/shopspring/decimal.Decimal"
          - db_type: "inet"
            go_type:
              import: "net/netip"
              type: "Addr"
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
```

---

## Schema Reference

| Artifact | Location | Contents |
|---|---|---|
| DDL + seed | `schema.sql` | 32 domain tables, 36 GPS partitions, 15 enum types |
| sqlc queries | `sql/queries/` | 251 named queries across 33 per-table files |
| Generated Go | `internal/db/` | Auto-generated — do not edit |

When there is a conflict between this file and the SQL files: SQL files win for data structure; this file wins for rules and behaviour.

---

*Repository: `petrosync` — Go API + HTMX Dashboard*
*Companion: `petrosync-android` — see that repo's SKILL.md for Android rules*
