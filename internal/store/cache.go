package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shiv6146/blayzen-sip/internal/models"
	"github.com/valkey-io/valkey-go"
)

// Cache provides caching operations using Valkey
type Cache struct {
	client   valkey.Client
	routeTTL time.Duration
}

// NewCache creates a new cache instance
func NewCache(ctx context.Context, url, password string, db int, routeTTL time.Duration) (*Cache, error) {
	opts := valkey.ClientOption{
		InitAddress: []string{url},
		SelectDB:    db,
	}
	if password != "" {
		opts.Password = password
	}

	client, err := valkey.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey client: %w", err)
	}

	// Test connection
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		return nil, fmt.Errorf("failed to ping valkey: %w", err)
	}

	return &Cache{
		client:   client,
		routeTTL: routeTTL,
	}, nil
}

// Close closes the cache connection
func (c *Cache) Close() {
	c.client.Close()
}

// routeKey generates the cache key for a route lookup
func routeKey(toUser, fromUser string) string {
	return fmt.Sprintf("route:%s:%s", toUser, fromUser)
}

// CacheRoutes caches routes for a specific lookup
func (c *Cache) CacheRoutes(ctx context.Context, toUser, fromUser string, routes []*models.Route) error {
	key := routeKey(toUser, fromUser)

	data, err := json.Marshal(routes)
	if err != nil {
		return err
	}

	return c.client.Do(ctx,
		c.client.B().Set().Key(key).Value(string(data)).Ex(c.routeTTL).Build(),
	).Error()
}

// GetCachedRoutes retrieves cached routes
func (c *Cache) GetCachedRoutes(ctx context.Context, toUser, fromUser string) ([]*models.Route, error) {
	key := routeKey(toUser, fromUser)

	result, err := c.client.Do(ctx, c.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var routes []*models.Route
	if err := json.Unmarshal([]byte(result), &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

// InvalidateRouteCache invalidates all route cache entries
func (c *Cache) InvalidateRouteCache(ctx context.Context) error {
	// Get all route keys
	keys, err := c.client.Do(ctx, c.client.B().Keys().Pattern("route:*").Build()).AsStrSlice()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all route keys
	return c.client.Do(ctx, c.client.B().Del().Key(keys...).Build()).Error()
}

// activeCallKey generates the cache key for tracking active calls
func activeCallKey(callID string) string {
	return fmt.Sprintf("call:active:%s", callID)
}

// SetActiveCall marks a call as active in the cache
func (c *Cache) SetActiveCall(ctx context.Context, callID string, data map[string]string) error {
	key := activeCallKey(callID)

	// Store call data with 1 hour TTL (calls shouldn't last longer)
	args := make([]string, 0, len(data)*2)
	for k, v := range data {
		args = append(args, k, v)
	}

	if err := c.client.Do(ctx,
		c.client.B().Hset().Key(key).FieldValue().FieldValue(args[0], args[1]).Build(),
	).Error(); err != nil {
		return err
	}

	// Set TTL
	return c.client.Do(ctx,
		c.client.B().Expire().Key(key).Seconds(3600).Build(),
	).Error()
}

// GetActiveCall retrieves active call data
func (c *Cache) GetActiveCall(ctx context.Context, callID string) (map[string]string, error) {
	key := activeCallKey(callID)

	result, err := c.client.Do(ctx, c.client.B().Hgetall().Key(key).Build()).AsStrMap()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RemoveActiveCall removes a call from the active calls cache
func (c *Cache) RemoveActiveCall(ctx context.Context, callID string) error {
	key := activeCallKey(callID)
	return c.client.Do(ctx, c.client.B().Del().Key(key).Build()).Error()
}

// GetActiveCallCount returns the number of active calls
func (c *Cache) GetActiveCallCount(ctx context.Context) (int64, error) {
	keys, err := c.client.Do(ctx, c.client.B().Keys().Pattern("call:active:*").Build()).AsStrSlice()
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

