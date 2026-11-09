package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"api-gateway/gateway/internal/auth"
	"api-gateway/gateway/internal/ratelimit"
	"api-gateway/gateway/internal/router"
	"api-gateway/gateway/internal/telemetry"
)

type GatewayProxy struct {
	auth      *auth.Authenticator
	limiter   *ratelimit.Limiter
	router    *router.DynamicRouter
	collector *telemetry.Collector
	transport *http.Transport
}

func NewGatewayProxy(
	auth *auth.Authenticator,
	limiter *ratelimit.Limiter,
	router *router.DynamicRouter,
	collector *telemetry.Collector,
) *GatewayProxy {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableCompression:  false,
	}

	return &GatewayProxy{
		auth:      auth,
		limiter:   limiter,
		router:    router,
		collector: collector,
		transport: transport,
	}
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (p *GatewayProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Special health check bypass
	if r.URL.Path == "/health" || r.URL.Path == "/_health" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"gateway"}`))
		return
	}

	// 1. Extract API Key
	rawKey, err := auth.ExtractRawKey(r)
	if err != nil {
		p.respondError(w, http.StatusUnauthorized, "missing_api_key", "Authentication required. Provide key via Authorization: Bearer gw_live_...")
		p.recordMetric("", "", http.StatusUnauthorized, start)
		return
	}

	// 2. Validate Key
	meta, err := p.auth.Authenticate(r.Context(), rawKey)
	if err != nil {
		p.respondError(w, http.StatusUnauthorized, "invalid_api_key", err.Error())
		p.recordMetric("", "", http.StatusUnauthorized, start)
		return
	}

	// 3. Validate Scope
	if !auth.ValidateScope(meta.Scopes, r.Method, r.URL.Path) {
		p.respondError(w, http.StatusForbidden, "forbidden_scope", "API key does not have scope permission for this endpoint")
		p.recordMetric(meta.ID, "", http.StatusForbidden, start)
		return
	}

	// 4. Rate Limiting Check
	rlResult, err := p.limiter.Allow(r.Context(), meta.ID, meta.RateLimitPerMin, meta.RateLimitPerSec, meta.MonthlyQuota)
	if err != nil {
		// Logged in limiter, fail-open
	}

	// Standard Rate Limit Headers
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(meta.RateLimitPerMin))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(rlResult.Remaining, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rlResult.ResetSeconds, 10))

	if !rlResult.Allowed {
		w.Header().Set("Retry-After", strconv.FormatInt(rlResult.ResetSeconds, 10))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":         "too_many_requests",
			"message":       "Rate limit or monthly quota exceeded",
			"reason":        rlResult.Reason,
			"retry_after_s": rlResult.ResetSeconds,
		})
		p.recordMetric(meta.ID, "", http.StatusTooManyRequests, start)
		return
	}

	// 5. Match Dynamic Route
	route, _ := p.router.Match(r.URL.Path)

	// 6. Upstream Proxying
	revProxy := httputil.NewSingleHostReverseProxy(route.UpstreamURL)
	revProxy.Transport = p.transport

	originalDirector := revProxy.Director
	revProxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Strip prefix if configured
		if route.StripPrefix && route.PathPrefix != "/" {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, route.PathPrefix)
			if !strings.HasPrefix(req.URL.Path, "/") {
				req.URL.Path = "/" + req.URL.Path
			}
		}

		// Inject gateway metadata headers for upstream backend
		req.Header.Set("X-Consumer-Id", meta.UserID)
		req.Header.Set("X-Key-Id", meta.ID)
		req.Header.Set("X-RateLimit-Remaining", strconv.FormatInt(rlResult.Remaining, 10))

		// Client IP tracking
		if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	revProxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(rw).Encode(map[string]string{
			"error":   "bad_gateway",
			"message": fmt.Sprintf("Failed to reach upstream backend: %v", proxyErr),
		})
	}

	capturingWriter := &statusCapturingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default if WriteHeader not called
	}

	revProxy.ServeHTTP(capturingWriter, r)

	p.recordMetric(meta.ID, route.ID, capturingWriter.statusCode, start)
}

func (p *GatewayProxy) respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

func (p *GatewayProxy) recordMetric(keyID, routeID string, statusCode int, start time.Time) {
	if p.collector == nil {
		return
	}
	latency := time.Since(start).Milliseconds()
	p.collector.Record(telemetry.MetricEvent{
		Timestamp:  time.Now().UTC(),
		KeyID:      keyID,
		RouteID:    routeID,
		StatusCode: statusCode,
		LatencyMs:  latency,
	})
}

