package metrics

import (
	"testing"

	"github.com/marcelsud/webhook-inbox/routes"
	"github.com/stretchr/testify/assert"
)

func TestRedisCollector_NewRedisCollector(t *testing.T) {
	t.Run("creates collector successfully", func(t *testing.T) {
		// This test verifies that the collector can be created
		// It doesn't require Redis connection for the constructor
		loader := routes.NewLoader()

		// Note: In a real test, you would use a mock Redis client
		// For this unit test, we're just testing the constructor
		collector := NewRedisCollector(nil, loader)

		assert.NotNil(t, collector)
		assert.NotNil(t, collector.routesLoader)
	})
}

func TestMetrics_Struct(t *testing.T) {
	t.Run("metrics struct has all required fields", func(t *testing.T) {
		m := Metrics{
			QueueLengths: map[string]int64{
				"route1": 10,
				"route2": 5,
			},
			StatusCounts: map[string]int64{
				"pending":   100,
				"delivered": 50,
				"failed":    5,
			},
			Throughput: ThroughputMetrics{
				LastMinute:         10,
				LastFiveMinutes:    45,
				LastFifteenMinutes: 120,
			},
			Workers: map[string][]WorkerInfo{
				"route1": {
					{
						WorkerID: "worker-1",
						RouteID:  "route1",
						Status:   "idle",
					},
				},
			},
		}

		assert.NotNil(t, m.QueueLengths)
		assert.NotNil(t, m.StatusCounts)
		assert.NotNil(t, m.Workers)
		assert.Equal(t, int64(10), m.Throughput.LastMinute)
	})
}

func TestThroughputMetrics(t *testing.T) {
	t.Run("throughput metrics structure", func(t *testing.T) {
		tp := ThroughputMetrics{
			LastMinute:         5,
			LastFiveMinutes:    20,
			LastFifteenMinutes: 50,
		}

		assert.Equal(t, int64(5), tp.LastMinute)
		assert.Equal(t, int64(20), tp.LastFiveMinutes)
		assert.Equal(t, int64(50), tp.LastFifteenMinutes)
	})
}

func TestWorkerInfo(t *testing.T) {
	t.Run("worker info structure", func(t *testing.T) {
		worker := WorkerInfo{
			WorkerID: "worker-1",
			RouteID:  "user-events",
			Status:   "processing",
		}

		assert.Equal(t, "worker-1", worker.WorkerID)
		assert.Equal(t, "user-events", worker.RouteID)
		assert.Equal(t, "processing", worker.Status)
	})
}

// Integration test helpers
func setupTestRoutes(t *testing.T) *routes.Loader {
	loader := routes.NewLoader()

	// Create temporary routes.yaml for testing
	// In a real scenario, you would create a test file or use in-memory config
	return loader
}

func TestCollector_Interface(t *testing.T) {
	t.Run("RedisCollector implements Collector interface", func(t *testing.T) {
		var _ Collector = (*RedisCollector)(nil)
	})
}

// Note: Full integration tests that require Redis should be placed in
// redis_collector_integration_test.go with build tag "integration"
