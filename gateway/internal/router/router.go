package router

import (
	"context"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const RouteInvalidationChan = "channel:route_invalidation"

type RouteConfig struct {
	ID          string
	Name        string
	PathPrefix  string
	UpstreamURL *url.URL
	StripPrefix bool
	Timeout     time.Duration
}

type DynamicRouter struct {
	dbPool          *pgxpool.Pool
	redisClient     *redis.Client
	defaultUpstream *url.URL
	routesMu        sync.RWMutex
	routes          []RouteConfig
}

func NewDynamicRouter(dbPool *pgxpool.Pool, redisClient *redis.Client, defaultUpstreamStr string) (*DynamicRouter, error) {
	parsedDefault, err := url.Parse(defaultUpstreamStr)
	if err != nil {
		return nil, err
	}

	dr := &DynamicRouter{
		dbPool:          dbPool,
		redisClient:     redisClient,
		defaultUpstream: parsedDefault,
	}

	dr.reloadRoutes()

	if redisClient != nil {
		go dr.listenInvalidations()
	}

	return dr, nil
}

func (dr *DynamicRouter) reloadRoutes() {
	if dr.dbPool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := dr.dbPool.Query(ctx, `
		SELECT id, name, path_prefix, upstream_url, strip_prefix, timeout_ms 
		FROM routes 
		WHERE is_active = true 
		ORDER BY LENGTH(path_prefix) DESC
	`)
	if err != nil {
		log.Printf("[WARN] Failed to load routes from DB: %v", err)
		return
	}
	defer rows.Close()

	var newRoutes []RouteConfig
	for rows.Next() {
		var id, name, prefix, rawURL string
		var strip bool
		var timeoutMs int

		if err := rows.Scan(&id, &name, &prefix, &rawURL, &strip, &timeoutMs); err != nil {
			continue
		}

		parsed, err := url.Parse(rawURL)
		if err != nil {
			log.Printf("[WARN] Invalid upstream URL for route %s: %s", id, rawURL)
			continue
		}

		newRoutes = append(newRoutes, RouteConfig{
			ID:          id,
			Name:        name,
			PathPrefix:  prefix,
			UpstreamURL: parsed,
			StripPrefix: strip,
			Timeout:     time.Duration(timeoutMs) * time.Millisecond,
		})
	}

	dr.routesMu.Lock()
	dr.routes = newRoutes
	dr.routesMu.Unlock()

	log.Printf("[INFO] Dynamic routes reloaded: %d active routes", len(newRoutes))
}

func (dr *DynamicRouter) listenInvalidations() {
	pubsub := dr.redisClient.Subscribe(context.Background(), RouteInvalidationChan)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for range ch {
		log.Println("[INFO] Route reload signal received via Redis")
		dr.reloadRoutes()
	}
}

// Match finds the best matching route prefix, or returns default upstream
func (dr *DynamicRouter) Match(requestPath string) (RouteConfig, bool) {
	dr.routesMu.RLock()
	defer dr.routesMu.RUnlock()

	for _, r := range dr.routes {
		if strings.HasPrefix(requestPath, r.PathPrefix) {
			return r, true
		}
	}

	// Fallback to default
	return RouteConfig{
		ID:          "route_default",
		Name:        "Default Upstream",
		PathPrefix:  "/",
		UpstreamURL: dr.defaultUpstream,
		StripPrefix: false,
		Timeout:     5 * time.Second,
	}, false
}

