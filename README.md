# ⚡ API Key & Rate Limiting Gateway

> **Ultra-low-latency (< 2-5 ms) Reverse Proxy & Access Management Platform** for microservices and APIs.

Handles API Key authentication, safe SHA-256 storage, scope verification, sliding window rate limiting, monthly billing quotas, asynchronous telemetry analytics, and dynamic upstream routing.

---

## 🏛 Architecture & Request Flow

```
[ Client / SDK ]
       │
       │  1. Request with Header: Authorization: Bearer gw_live_...
       ▼
┌────────────────────────────────────────────────────────┐
│               API Gateway (Reverse Proxy)              │
│                                                        │
│  2. Fast Validation:                                   │
│     - Key Hashing (SHA-256)                            │
│     - L1 In-Memory Cache -> L2 Redis Cache             │
│     - Scope matching (e.g. "GET /api/v1/*")            │
│     - Atomic Redis Lua script (Sliding Window Counter) │
│                                                        │
│  3. If rate limit exceeded: HTTP 429 Too Many Requests │
│  4. Async Non-Blocking Telemetry: buffered batch flush │
└──────────────────┬─────────────────────────────────────┘
                   │
                   │  5. Upstream proxy with injected headers:
                   │     X-Consumer-Id: usr_123
                   │     X-Key-Id: key_abc
                   │     X-RateLimit-Remaining: 42
                   ▼
         [ Your Upstream Target Backend ]
```

---

## 🚀 Key Features

1. **API Key Management (`gw_live_...`)**
   - Cryptographically random keys with recognizable prefix (compatible with GitHub Secret Scanning).
   - Only salted SHA-256 hashes are persisted in PostgreSQL.
   - Secret key revealed **strictly once** upon generation.
   - 1-click revocation with real-time Redis Pub/Sub invalidation.

2. **Ultra-Fast Rate Limiter (Redis Lua Script in 1 RTT)**
   - **Sliding Window Counter**: Smooth, memory-efficient rate limiting combining current and previous windows without the ballooning memory footprint of ZSET logs.
   - **Per-Second Burst Protection**: Guard against spikes and DoS.
   - **Monthly Billing Quota**: Enforces plan thresholds (Free, Starter, Pro).
   - **RFC Standard Headers**:
     - `X-RateLimit-Limit: 60`
     - `X-RateLimit-Remaining: 42`
     - `X-RateLimit-Reset: 18`

3. **High-Performance Data Plane**
   - Built with Go `net/http` and `httputil.ReverseProxy` with tuned connection pools (`MaxIdleConns: 1000`, `IdleConnTimeout: 90s`).
   - L1 in-memory cache + L2 Redis cache for sub-millisecond key lookups.
   - Telemetry collector uses buffered channels and asynchronous worker pools to batch-flush metrics to PostgreSQL every 2 seconds without delaying proxy requests.

4. **Full-Featured Admin Dashboard**
   - Modern Next.js 15 + Tailwind CSS interface.
   - Live traffic overview (RPS, success rate, 429 blocks, latency).
   - API key manager with one-time copy modal.
   - Tiers & Quotas editor (Free, Starter, Pro).
   - Upstream services & routes manager with prefix stripping.
   - **Interactive Live Playground**: Test HTTP calls and run 15x parallel bursts to see rate limiting and headers live!

---

## 🛠 Quickstart with Docker Compose

```bash
# 1. Clone repository & configure environment
cp .env.example .env

# 2. Build and launch all services
docker compose up --build -d

# 3. Access Services:
# - Admin Dashboard:  http://localhost:3000
# - API Gateway:      http://localhost:8000
# - Control Plane:    http://localhost:8081
# - Mock Backend:     http://localhost:8080
```

---

## 💻 Manual / Local Development

### 1. Build Binaries
```bash
go build -o gateway/gateway.exe ./gateway/cmd/gateway
go build -o control-plane/control-plane.exe ./control-plane/cmd/server
go build -o mock-backend/mock-backend.exe ./mock-backend
```

### 2. Run Tests
```bash
go test -v ./gateway/... ./control-plane/...
```

### 3. Run Next.js Dashboard
```bash
cd dashboard
npm install
npm run dev
```

---

## 📡 API Example Usage

### 1. Make Request through the Gateway
```bash
curl -i http://localhost:8000/mock/api/v1/users \
  -H "Authorization: Bearer gw_live_YOUR_SECRET_KEY"
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 59
X-RateLimit-Reset: 45

{
  "status": "success",
  "data": [
    {"id": "usr_1", "name": "Alice Developer", "tier": "Pro"}
  ]
}
```

### 2. Rate Limit Exceeded Response (429)
```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 14
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 14

{
  "error": "too_many_requests",
  "message": "Rate limit or monthly quota exceeded",
  "reason": "RATE_LIMIT_EXCEEDED",
  "retry_after_s": 14
}
```

