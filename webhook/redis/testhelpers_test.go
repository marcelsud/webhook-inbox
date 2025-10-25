//go:build integration

package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/marcelsud/webhook-inbox/webhook/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	testcontainersredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

/* Test Helpers for Redis Integration Tests
 * Following the pattern from: https://eltonminetto.dev/post/2024-02-15-using-test-helpers/
 */

// RedisContainer holds the Redis testcontainer and connection details
type RedisContainer struct {
	Container *testcontainersredis.RedisContainer
	Addr      string
}

// SetupRedisContainer creates and starts a Redis testcontainer
func SetupRedisContainer(t *testing.T, ctx context.Context) (*RedisContainer, func()) {
	t.Helper()

	// Start Redis container
	redisContainer, err := testcontainersredis.Run(ctx,
		"redis:7-alpine",
		testcontainersredis.WithSnapshotting(10, 1),
		testcontainersredis.WithLogLevel(testcontainersredis.LogLevelVerbose),
	)
	require.NoError(t, err, "failed to start Redis container")

	// Get connection string
	addr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err, "failed to get Redis connection string")

	// Remove redis:// prefix if present
	if len(addr) > 8 && addr[:8] == "redis://" {
		addr = addr[8:]
	}

	// Wait for Redis to be ready
	time.Sleep(1 * time.Second)

	rc := &RedisContainer{
		Container: redisContainer,
		Addr:      addr,
	}

	// Cleanup function
	cleanup := func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate Redis container: %v", err)
		}
	}

	return rc, cleanup
}

// CreateTestRepository creates a Redis repository connected to the test container
func CreateTestRepository(t *testing.T, addr string) *redis.Repository {
	t.Helper()

	repo, err := redis.NewRepository(addr, "", 0)
	require.NoError(t, err, "failed to create Redis repository")

	return repo
}

// GenerateID is a helper to generate test webhook IDs
func GenerateID(t *testing.T, index int) string {
	t.Helper()
	return fmt.Sprintf("test-webhook-%d-%d", index, time.Now().UnixNano())
}

// GetKeyTTL returns the TTL of a Redis key in seconds
func GetKeyTTL(t *testing.T, addr string, key string) int64 {
	t.Helper()

	client := createRedisClient(addr)
	defer client.Close()

	ttl, err := client.TTL(context.Background(), key).Result()
	require.NoError(t, err)

	return int64(ttl.Seconds())
}

// KeyExists checks if a Redis key exists
func KeyExists(t *testing.T, addr string, key string) bool {
	t.Helper()

	client := createRedisClient(addr)
	defer client.Close()

	exists, err := client.Exists(context.Background(), key).Result()
	require.NoError(t, err)

	return exists > 0
}

// createRedisClient creates a direct Redis client for testing helpers
func createRedisClient(addr string) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})
}
