package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// WorkerHeartbeat represents the heartbeat data for a worker
type WorkerHeartbeat struct {
	WorkerID      string    `json:"worker_id"`
	RouteID       string    `json:"route_id"`
	Status        string    `json:"status"` // "idle", "processing"
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// SetWorkerHeartbeat stores or updates a worker's heartbeat in Redis
// The heartbeat key has a TTL of 60 seconds - if a worker doesn't send a heartbeat
// within that time, it's considered inactive
func (r *Repository) SetWorkerHeartbeat(ctx context.Context, workerID, routeID, status string) error {
	key := fmt.Sprintf("worker:heartbeat:%s:%s", routeID, workerID)

	heartbeat := WorkerHeartbeat{
		WorkerID:      workerID,
		RouteID:       routeID,
		Status:        status,
		LastHeartbeat: time.Now(),
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		return fmt.Errorf("marshaling heartbeat: %w", err)
	}

	// Set with 60 second TTL - workers should send heartbeats every 30 seconds
	err = r.client.Set(ctx, key, data, 60*time.Second).Err()
	if err != nil {
		return fmt.Errorf("setting heartbeat: %w", err)
	}

	return nil
}

// GetActiveWorkers retrieves all active workers for a given route
func (r *Repository) GetActiveWorkers(ctx context.Context, routeID string) ([]WorkerHeartbeat, error) {
	pattern := fmt.Sprintf("worker:heartbeat:%s:*", routeID)
	var workers []WorkerHeartbeat

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scanning worker keys: %w", err)
		}

		for _, key := range keys {
			data, err := r.client.Get(ctx, key).Result()
			if err == redis.Nil {
				// Key expired between scan and get
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("getting worker heartbeat: %w", err)
			}

			var heartbeat WorkerHeartbeat
			if err := json.Unmarshal([]byte(data), &heartbeat); err != nil {
				continue
			}

			workers = append(workers, heartbeat)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return workers, nil
}

// GetAllActiveWorkers retrieves all active workers across all routes
func (r *Repository) GetAllActiveWorkers(ctx context.Context) (map[string][]WorkerHeartbeat, error) {
	pattern := "worker:heartbeat:*"
	workersByRoute := make(map[string][]WorkerHeartbeat)

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scanning worker keys: %w", err)
		}

		for _, key := range keys {
			data, err := r.client.Get(ctx, key).Result()
			if err == redis.Nil {
				// Key expired between scan and get
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("getting worker heartbeat: %w", err)
			}

			var heartbeat WorkerHeartbeat
			if err := json.Unmarshal([]byte(data), &heartbeat); err != nil {
				continue
			}

			workersByRoute[heartbeat.RouteID] = append(workersByRoute[heartbeat.RouteID], heartbeat)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return workersByRoute, nil
}
