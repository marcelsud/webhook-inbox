package metrics

import (
	"context"
	"time"
)

// Metrics represents the current state of the webhook system.
type Metrics struct {
	// QueueLengths maps route_id to the number of pending webhooks in the queue
	QueueLengths map[string]int64 `json:"queue_lengths"`

	// StatusCounts maps status name to count of webhooks in that status
	StatusCounts map[string]int64 `json:"status_counts"`

	// Throughput represents webhooks processed per time window
	Throughput ThroughputMetrics `json:"throughput"`

	// Workers maps route_id to list of active workers
	Workers map[string][]WorkerInfo `json:"workers"`

	// Timestamp when metrics were collected
	Timestamp time.Time `json:"timestamp"`
}

// ThroughputMetrics represents webhooks processed over different time windows.
type ThroughputMetrics struct {
	// LastMinute is webhooks delivered in the last 1 minute
	LastMinute int64 `json:"last_minute"`

	// LastFiveMinutes is webhooks delivered in the last 5 minutes
	LastFiveMinutes int64 `json:"last_five_minutes"`

	// LastFifteenMinutes is webhooks delivered in the last 15 minutes
	LastFifteenMinutes int64 `json:"last_fifteen_minutes"`
}

// WorkerInfo represents information about an active worker.
type WorkerInfo struct {
	// WorkerID is a unique identifier for the worker
	WorkerID string `json:"worker_id"`

	// RouteID is the route this worker is processing
	RouteID string `json:"route_id"`

	// Status is the current status of the worker (e.g., "idle", "processing")
	Status string `json:"status"`

	// LastHeartbeat is the timestamp of the last heartbeat
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// Collector defines the interface for collecting metrics from the webhook system.
type Collector interface {
	// Collect gathers current metrics from the system
	Collect(ctx context.Context) (Metrics, error)

	// GetQueueLengths returns the number of pending webhooks per route
	GetQueueLengths(ctx context.Context) (map[string]int64, error)

	// GetStatusCounts returns the count of webhooks by status
	GetStatusCounts(ctx context.Context) (map[string]int64, error)

	// GetThroughput returns webhooks processed over time windows
	GetThroughput(ctx context.Context) (ThroughputMetrics, error)

	// GetActiveWorkers returns information about active workers per route
	GetActiveWorkers(ctx context.Context) (map[string][]WorkerInfo, error)
}
