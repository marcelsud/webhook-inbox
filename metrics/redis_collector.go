package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/marcelsud/webhook-inbox/routes"
	"github.com/redis/go-redis/v9"
)

// RedisCollector implements the Collector interface for Redis-backed metrics
type RedisCollector struct {
	client       *redis.Client
	routesLoader *routes.Loader
}

// NewRedisCollector creates a new Redis metrics collector
func NewRedisCollector(client *redis.Client, loader *routes.Loader) *RedisCollector {
	return &RedisCollector{
		client:       client,
		routesLoader: loader,
	}
}

// Collect gathers all metrics from Redis
func (c *RedisCollector) Collect(ctx context.Context) (Metrics, error) {
	queueLengths, err := c.GetQueueLengths(ctx)
	if err != nil {
		return Metrics{}, fmt.Errorf("getting queue lengths: %w", err)
	}

	statusCounts, err := c.GetStatusCounts(ctx)
	if err != nil {
		return Metrics{}, fmt.Errorf("getting status counts: %w", err)
	}

	throughput, err := c.GetThroughput(ctx)
	if err != nil {
		return Metrics{}, fmt.Errorf("getting throughput: %w", err)
	}

	workers, err := c.GetActiveWorkers(ctx)
	if err != nil {
		return Metrics{}, fmt.Errorf("getting active workers: %w", err)
	}

	return Metrics{
		QueueLengths: queueLengths,
		StatusCounts: statusCounts,
		Throughput:   throughput,
		Workers:      workers,
		Timestamp:    time.Now(),
	}, nil
}

// GetQueueLengths returns the number of pending webhooks in each stream
func (c *RedisCollector) GetQueueLengths(ctx context.Context) (map[string]int64, error) {
	queueLengths := make(map[string]int64)
	allRoutes := c.routesLoader.List()

	for _, route := range allRoutes {
		streamKey := fmt.Sprintf("webhooks:%s:%s", route.Mode.String(), route.RouteID)

		length, err := c.client.XLen(ctx, streamKey).Result()
		if err != nil && err != redis.Nil {
			// Continue even if one stream fails
			continue
		}

		queueLengths[route.RouteID] = length
	}

	return queueLengths, nil
}

// GetStatusCounts returns counts of webhooks grouped by status
func (c *RedisCollector) GetStatusCounts(ctx context.Context) (map[string]int64, error) {
	statusCounts := map[string]int64{
		"pending":    0,
		"delivering": 0,
		"delivered":  0,
		"failed":     0,
		"retrying":   0,
	}

	// Scan for all webhook:* keys
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = c.client.Scan(ctx, cursor, "webhook:*", 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("scanning webhook keys: %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	// Filter out message ID keys (webhook:*:msgid)
	var webhookKeys []string
	for _, key := range keys {
		// Skip message ID keys
		if len(key) > 6 && key[len(key)-6:] == ":msgid" {
			continue
		}
		webhookKeys = append(webhookKeys, key)
	}

	// Batch get status for all webhooks
	if len(webhookKeys) == 0 {
		return statusCounts, nil
	}

	// Use pipeline for efficient batch operations
	pipe := c.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(webhookKeys))

	for i, key := range webhookKeys {
		cmds[i] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("executing pipeline: %w", err)
	}

	// Count statuses
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil || len(data) == 0 {
			continue
		}

		status := data["status"]
		if _, exists := statusCounts[status]; exists {
			statusCounts[status]++
		}
	}

	return statusCounts, nil
}

// GetThroughput calculates webhooks delivered over different time windows
func (c *RedisCollector) GetThroughput(ctx context.Context) (ThroughputMetrics, error) {
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute).Unix()
	fiveMinutesAgo := now.Add(-5 * time.Minute).Unix()
	fifteenMinutesAgo := now.Add(-15 * time.Minute).Unix()

	var lastMinute, lastFiveMinutes, lastFifteenMinutes int64

	// Scan for all webhook:* keys (excluding msgid keys)
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "webhook:*", 1000).Result()
		if err != nil {
			return ThroughputMetrics{}, fmt.Errorf("scanning webhook keys: %w", err)
		}

		// Filter and process keys
		for _, key := range keys {
			// Skip message ID keys
			if len(key) > 6 && key[len(key)-6:] == ":msgid" {
				continue
			}

			// Get status and updated_at
			data, err := c.client.HMGet(ctx, key, "status", "updated_at").Result()
			if err != nil || len(data) < 2 {
				continue
			}

			status, ok1 := data[0].(string)
			updatedAtStr, ok2 := data[1].(string)
			if !ok1 || !ok2 || status != "delivered" {
				continue
			}

			var updatedAt int64
			fmt.Sscanf(updatedAtStr, "%d", &updatedAt)

			// Count in time windows
			if updatedAt >= fifteenMinutesAgo {
				lastFifteenMinutes++
				if updatedAt >= fiveMinutesAgo {
					lastFiveMinutes++
					if updatedAt >= oneMinuteAgo {
						lastMinute++
					}
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return ThroughputMetrics{
		LastMinute:         lastMinute,
		LastFiveMinutes:    lastFiveMinutes,
		LastFifteenMinutes: lastFifteenMinutes,
	}, nil
}

// GetActiveWorkers returns information about active workers
func (c *RedisCollector) GetActiveWorkers(ctx context.Context) (map[string][]WorkerInfo, error) {
	workers := make(map[string][]WorkerInfo)

	// Scan for worker:heartbeat:* keys
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "worker:heartbeat:*", 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("scanning worker heartbeat keys: %w", err)
		}

		for _, key := range keys {
			// Get worker heartbeat data
			data, err := c.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var workerInfo WorkerInfo
			if err := json.Unmarshal([]byte(data), &workerInfo); err != nil {
				continue
			}

			// Group by route ID
			workers[workerInfo.RouteID] = append(workers[workerInfo.RouteID], workerInfo)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return workers, nil
}
