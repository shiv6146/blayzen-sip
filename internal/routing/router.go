// Package routing handles SIP call routing logic
package routing

import (
	"context"
	"fmt"

	"github.com/shiv6146/blayzen-sip/internal/models"
	"github.com/shiv6146/blayzen-sip/internal/store"
)

// Router handles inbound call routing
type Router struct {
	store          *store.PostgresStore
	cache          *store.Cache
	defaultWSURL   string
}

// NewRouter creates a new routing engine
func NewRouter(store *store.PostgresStore, cache *store.Cache, defaultWSURL string) *Router {
	return &Router{
		store:        store,
		cache:        cache,
		defaultWSURL: defaultWSURL,
	}
}

// FindRoute finds the best matching route for an inbound call
func (r *Router) FindRoute(ctx context.Context, toUser, fromUser string, headers map[string]string) (*models.Route, error) {
	// Try cache first
	var routes []*models.Route
	var err error

	if r.cache != nil {
		routes, err = r.cache.GetCachedRoutes(ctx, toUser, fromUser)
		if err != nil {
			// Log but don't fail - fall back to database
			routes = nil
		}
	}

	// If not in cache, query database
	if routes == nil {
		routes, err = r.store.FindMatchingRoutes(ctx, toUser, fromUser)
		if err != nil {
			return nil, fmt.Errorf("failed to find routes: %w", err)
		}

		// Cache the results
		if r.cache != nil && len(routes) > 0 {
			_ = r.cache.CacheRoutes(ctx, toUser, fromUser, routes)
		}
	}

	// Find best match considering custom headers
	for _, route := range routes {
		if route.Matches(toUser, fromUser, headers) {
			return route, nil
		}
	}

	// No specific route found, use default if available
	if r.defaultWSURL != "" {
		return &models.Route{
			Name:         "default",
			WebSocketURL: r.defaultWSURL,
		}, nil
	}

	return nil, fmt.Errorf("no matching route found for to=%s from=%s", toUser, fromUser)
}

// InvalidateCache invalidates the routing cache
func (r *Router) InvalidateCache(ctx context.Context) error {
	if r.cache != nil {
		return r.cache.InvalidateRouteCache(ctx)
	}
	return nil
}

