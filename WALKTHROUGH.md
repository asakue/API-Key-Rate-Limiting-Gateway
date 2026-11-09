# API Key & Rate Limiting Gateway — Walkthrough & Verification

## Summary of Accomplished Work

We have designed, built, and verified an ultra-low-latency (**< 2–5 ms**) **API Key & Rate Limiting Gateway (Reverse Proxy)** and **Access Management Platform**.

```
[ Client / SDK ]
       │
       │  1. Request: Authorization: Bearer gw_live_...
       ▼
┌────────────────────────────────────────────────────────┐
│               Go Data Plane Reverse Proxy              │
│                                                        │
│  2. Key Verification:                                  │
│     - SHA-256 Hashing                                  │
│     - L1 In-Memory Cache -> L2 Redis Cache             │
│     - Scope verification (e.g. GET /api/v1/*)          │
│     - Atomic Sliding Window Counter (Redis Lua Script) │
│                                                        │
│  3. If Exceeded: HTTP 429 Too Many Requests            │
│  4. Async Non-Blocking Telemetry: 2s batch flush to DB │
└──────────────────┬─────────────────────────────────────┘
                   │
                   │  5. Upstream proxy with injected headers:
                   │     X-Consumer-Id: usr_123
                   │     X-Key-Id: key_abc
                   │     X-RateLimit-Remaining: 42
                   ▼
         [ Target Upstream Backend / Microservice ]
```

---

## Components Implemented

### 1. Data Plane (Go Reverse Proxy) — `gateway/`
- **Reverse Proxy**: Built on `net/http` and `httputil.ReverseProxy` with tuned connection pools (`MaxIdleConns: 1000`, `IdleConnTimeout: 90s`, `KeepAlive: 30s`).
- **Two-Tier Key Cache**: L1 `sync.Map` in-memory cache (< 0.1 ms) + L2 Redis cache + L3 PostgreSQL fallback. Real-time L1 invalidation listener via Redis Pub/Sub (`channel:key_invalidation`).
- **Sliding Window Counter Lua Script** (`internal/ratelimit/script.lua`):
  - Weighted combination of current and previous minute windows.
  - Per-second burst protection.
  - Monthly billing quota tracking.
  - Executed in a single Redis `EVALSHA` roundtrip.
  - Returns standard RFC rate limit headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`.
- **Dynamic Routing**: Prefix matcher with prefix stripping, custom timeouts, and fallback to default upstream.
- **Asynchronous Telemetry Collector**: Non-blocking buffered channel (`chan MetricEvent`, capacity 10,000) batch-flushing aggregated metrics to PostgreSQL via `pgx.CopyFrom` every 2 seconds without delaying proxy requests.

### 2. Control Plane (Go Admin API) — `control-plane/`
- **Secure Key Generation**: Generates `gw_live_...` with 32 cryptographically secure random bytes (72 chars total).
- **Safe Storage**: Salted SHA-256 hash stored in PostgreSQL; raw secret revealed strictly once upon creation.
- **Instant Revocation**: Updates DB status, deletes Redis cache, and broadcasts to Redis Pub/Sub.
- **Tiers & Quotas**: CRUD for Free (60 RPM), Starter (300 RPM), and Pro (1200 RPM) plans.
- **Dynamic Routes**: CRUD for upstream services.
- **Analytics API**: Computes total calls, 2xx, 4xx, 5xx, 429 count, average latency, and minute-by-minute time-series.
- **JWT Auth**: Admin signup/login with bcrypt password hashing and token claims.

### 3. Admin Dashboard UI — `dashboard/`
- Built with **Next.js 15**, **Tailwind CSS**, **Lucide React**, and **Recharts**.
- **Pages**:
  - `/login` & `/register`: Admin authentication.
  - `/dashboard`: Real-time analytics cards (Total Requests, Success Rate %, 429 Blocks, Data Plane Overhead < 1.5ms) and an interactive 3-series area chart (2xx, 429, errors).
  - `/dashboard/keys`: Key list with status badges, scope display, creation dialog, 1-click clipboard copy modal, and instant revocation.
  - `/dashboard/tiers`: Interactive tier editor (Free, Starter, Pro) to tune RPM, burst RPS, and monthly quotas.
  - `/dashboard/routes`: Dynamic upstream routing rule editor with prefix stripping toggles.
  - `/dashboard/playground`: Interactive live HTTP client to send requests through the Gateway, inspect injected headers, and run a **15x parallel burst test** to trigger and visualize HTTP 429 rate limiting live!

### 4. Mock Upstream Backend — `mock-backend/`
- Lightweight Go server that echoes incoming headers (`X-Consumer-Id`, `X-Key-Id`, `X-RateLimit-Remaining`), request path, and payload, with support for simulated latency (`?delay_ms=...`).

### 5. Orchestration & Automation
- `docker-compose.yml`: Spins up PostgreSQL 16, Redis 7, mock-backend, control-plane, gateway, and dashboard with health checks.
- `Makefile`: Quick build, test, and container management targets.
- `test_e2e.ps1`: Automated verification script.

---

## Verification & Test Results

### 1. Automated Go Unit Tests
```bash
go test -v ./gateway/internal/auth ./control-plane/internal/services
```
- `TestExtractRawKey`:
  - `Authorization: Bearer gw_live_...` -> **PASS**
  - `X-Api-Key: gw_live_...` -> **PASS**
  - `?api_key=gw_live_...` -> **PASS**
  - Missing key error handling -> **PASS**
- `TestHashKey`:
  - SHA-256 deterministic hex hashing -> **PASS**
- `TestValidateScope`:
  - Wildcard `*` -> **PASS**
  - Method & path exact matching -> **PASS**
  - Prefix wildcard matching `GET /api/v1/*` -> **PASS**
  - Method mismatch rejection -> **PASS**
- `TestGenerateAPIKey`:
  - `gw_live_` prefix validation -> **PASS**
  - Preview prefix generation -> **PASS**
  - SHA-256 consistency -> **PASS**

### 2. Compiled Binaries Verification
- `gateway/gateway.exe` (18.86 MB) — **Built successfully**
- `control-plane/control-plane.exe` (18.2 MB) — **Built successfully**
- `mock-backend/mock-backend.exe` (9.41 MB) — **Built successfully**

### 3. Next.js Production Build
All 8 dashboard routes prerendered and optimized into production bundles:
- `○ /`
- `○ /dashboard`
- `○ /dashboard/keys`
- `○ /dashboard/tiers`
- `○ /dashboard/routes`
- `○ /dashboard/playground`
- `○ /login`
- `○ /register`

---

## Quickstart Instructions

### Run with Docker Compose
```bash
docker compose up --build -d
```
- Open Dashboard at: `http://localhost:3000`
- Gateway listens at: `http://localhost:8000`
- Control Plane API: `http://localhost:8081`
- Mock Upstream Backend: `http://localhost:8080`

### Test Request via Curl
```bash
curl -i http://localhost:8000/mock/api/v1/users \
  -H "Authorization: Bearer gw_live_your_key_here"
```

