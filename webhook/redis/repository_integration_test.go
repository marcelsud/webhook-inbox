//go:build integration

package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Store_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("store webhook in Redis", func(t *testing.T) {
		// Setup Redis container
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Create test webhook
		wh := webhook.Webhook{
			ID:           "test-webhook-1",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "data"}`),
			Headers:      map[string]string{"Content-Type": "application/json"},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Store webhook
		id, err := repo.Store(ctx, wh)

		require.NoError(t, err)
		assert.Equal(t, wh.ID, id)
	})

	t.Run("store and retrieve webhook", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		wh := webhook.Webhook{
			ID:           "test-webhook-2",
			RouteID:      "analytics",
			Payload:      []byte(`{"event": "user.created", "user_id": 123}`),
			Headers:      map[string]string{"X-Event-Type": "user.created"},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   5,
			DeliveryMode: webhook.PubSub,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Store
		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Retrieve
		retrieved, err := repo.Get(ctx, wh.ID)
		require.NoError(t, err)

		assert.Equal(t, wh.ID, retrieved.ID)
		assert.Equal(t, wh.RouteID, retrieved.RouteID)
		assert.Equal(t, string(wh.Payload), string(retrieved.Payload))
		assert.Equal(t, wh.Status, retrieved.Status)
		assert.Equal(t, wh.RetryCount, retrieved.RetryCount)
		assert.Equal(t, wh.MaxRetries, retrieved.MaxRetries)
		assert.Equal(t, wh.DeliveryMode, retrieved.DeliveryMode)
		assert.Equal(t, "user.created", retrieved.Headers["X-Event-Type"])
	})
}

func TestRepository_UpdateStatus_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("update webhook status", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Create and store webhook
		wh := webhook.Webhook{
			ID:           "test-webhook-3",
			RouteID:      "orders",
			Payload:      []byte(`{"order_id": 456}`),
			Headers:      map[string]string{},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Update status
		err = repo.UpdateStatus(ctx, wh.ID, webhook.Delivering)
		require.NoError(t, err)

		// Verify
		retrieved, err := repo.Get(ctx, wh.ID)
		require.NoError(t, err)
		assert.Equal(t, webhook.Delivering, retrieved.Status)

		// Update to delivered
		err = repo.UpdateStatus(ctx, wh.ID, webhook.Delivered)
		require.NoError(t, err)

		retrieved, err = repo.Get(ctx, wh.ID)
		require.NoError(t, err)
		assert.Equal(t, webhook.Delivered, retrieved.Status)
	})
}

func TestRepository_IncrementRetry_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("increment retry count", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		wh := webhook.Webhook{
			ID:           "test-webhook-4",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "retry"}`),
			Headers:      map[string]string{},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Increment retry count multiple times
		err = repo.IncrementRetry(ctx, wh.ID)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, wh.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, retrieved.RetryCount)

		err = repo.IncrementRetry(ctx, wh.ID)
		require.NoError(t, err)

		retrieved, err = repo.Get(ctx, wh.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.RetryCount)

		err = repo.IncrementRetry(ctx, wh.ID)
		require.NoError(t, err)

		retrieved, err = repo.Get(ctx, wh.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, retrieved.RetryCount)
	})
}

func TestRepository_Consume_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("consume FIFO webhooks", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		routeID := "fifo-route"

		// Store multiple webhooks
		for i := 1; i <= 3; i++ {
			wh := webhook.Webhook{
				ID:           string(rune('a'+i-1)) + "-webhook",
				RouteID:      routeID,
				Payload:      []byte(`{"order": ` + string(rune('0'+i)) + `}`),
				Headers:      map[string]string{},
				Status:       webhook.Pending,
				RetryCount:   0,
				MaxRetries:   3,
				DeliveryMode: webhook.FIFO,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Consume webhooks (should get them in order)
		webhooks, err := repo.Consume(ctx, routeID, webhook.FIFO)
		require.NoError(t, err)
		require.Len(t, webhooks, 1) // XREADGROUP returns one at a time by default

		// Verify we got a webhook
		assert.NotEmpty(t, webhooks[0].ID)
		assert.Equal(t, routeID, webhooks[0].RouteID)
	})

	t.Run("consume PubSub webhooks", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		routeID := "pubsub-route"

		// Store webhook
		wh := webhook.Webhook{
			ID:           "pubsub-webhook-1",
			RouteID:      routeID,
			Payload:      []byte(`{"event": "analytics"}`),
			Headers:      map[string]string{},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.PubSub,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Consume
		webhooks, err := repo.Consume(ctx, routeID, webhook.PubSub)
		require.NoError(t, err)
		require.Len(t, webhooks, 1)

		assert.Equal(t, wh.ID, webhooks[0].ID)
		assert.Equal(t, webhook.PubSub, webhooks[0].DeliveryMode)
	})

	t.Run("consume returns empty when no webhooks", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Try to consume from empty stream
		webhooks, err := repo.Consume(ctx, "empty-route", webhook.FIFO)
		require.NoError(t, err)
		assert.Empty(t, webhooks)
	})
}

func TestRepository_Acknowledge_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("acknowledge webhook", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		routeID := "ack-route"
		wh := webhook.Webhook{
			ID:           "ack-webhook-1",
			RouteID:      routeID,
			Payload:      []byte(`{"test": "ack"}`),
			Headers:      map[string]string{},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Consume webhook
		webhooks, err := repo.Consume(ctx, routeID, webhook.FIFO)
		require.NoError(t, err)
		require.Len(t, webhooks, 1)

		// Acknowledge it
		err = repo.Acknowledge(ctx, routeID, webhook.FIFO, wh.ID)
		require.NoError(t, err)

		// Try to consume again - should get nothing (already acknowledged)
		webhooks, err = repo.Consume(ctx, routeID, webhook.FIFO)
		require.NoError(t, err)
		assert.Empty(t, webhooks)
	})
}

func TestRepository_MultipleWebhooks_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("store and consume multiple webhooks", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		routeID := "multi-route"
		numWebhooks := 5

		// Store multiple webhooks
		for i := 0; i < numWebhooks; i++ {
			wh := webhook.Webhook{
				ID:           webhook.GenerateID(t, i),
				RouteID:      routeID,
				Payload:      []byte(`{"index": ` + string(rune('0'+i)) + `}`),
				Headers:      map[string]string{},
				Status:       webhook.Pending,
				RetryCount:   0,
				MaxRetries:   3,
				DeliveryMode: webhook.FIFO,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Consume all webhooks
		consumedCount := 0
		for i := 0; i < numWebhooks; i++ {
			webhooks, err := repo.Consume(ctx, routeID, webhook.FIFO)
			require.NoError(t, err)

			if len(webhooks) > 0 {
				consumedCount++
				// Acknowledge each webhook
				err = repo.Acknowledge(ctx, routeID, webhook.FIFO, webhooks[0].ID)
				require.NoError(t, err)
			}
		}

		assert.Equal(t, numWebhooks, consumedCount)
	})
}

func TestRepository_ErrorCases_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("get non-existent webhook", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		_, err := repo.Get(ctx, "non-existent-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update status of non-existent webhook", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Should not error (Redis HSET creates if not exists)
		err := repo.UpdateStatus(ctx, "non-existent", webhook.Delivered)
		require.NoError(t, err)
	})
}

func TestRepository_TTL_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("set TTL on webhook hash", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Create and store webhook
		wh := webhook.Webhook{
			ID:           "ttl-webhook-1",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "ttl"}`),
			Headers:      map[string]string{},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Set TTL of 5 seconds
		err = repo.SetTTL(ctx, wh.ID, 5*time.Second)
		require.NoError(t, err)

		// Verify webhook still exists immediately
		retrieved, err := repo.Get(ctx, wh.ID)
		require.NoError(t, err)
		assert.Equal(t, wh.ID, retrieved.ID)

		// Verify TTL is set using Redis client directly
		ttl := GetKeyTTL(t, redisContainer.Addr, "webhook:"+wh.ID)
		assert.Greater(t, ttl, int64(0), "TTL should be set")
		assert.LessOrEqual(t, ttl, int64(5), "TTL should be <= 5 seconds")
	})

	t.Run("webhook expires after TTL", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Create webhook
		wh := webhook.Webhook{
			ID:           "ttl-webhook-2",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "expire"}`),
			Headers:      map[string]string{},
			Status:       webhook.Delivered,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Set very short TTL (2 seconds)
		err = repo.SetTTL(ctx, wh.ID, 2*time.Second)
		require.NoError(t, err)

		// Verify it exists initially
		_, err = repo.Get(ctx, wh.ID)
		require.NoError(t, err)

		// Wait for TTL to expire
		time.Sleep(3 * time.Second)

		// Should not exist anymore
		_, err = repo.Get(ctx, wh.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("different TTLs for delivered and failed", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Create delivered webhook
		whDelivered := webhook.Webhook{
			ID:           "ttl-delivered",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "delivered"}`),
			Headers:      map[string]string{},
			Status:       webhook.Delivered,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err := repo.Store(ctx, whDelivered)
		require.NoError(t, err)

		// Create failed webhook
		whFailed := webhook.Webhook{
			ID:           "ttl-failed",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "failed"}`),
			Headers:      map[string]string{},
			Status:       webhook.Failed,
			RetryCount:   3,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err = repo.Store(ctx, whFailed)
		require.NoError(t, err)

		// Set different TTLs (delivered: 1 hour, failed: 24 hours)
		err = repo.SetTTL(ctx, whDelivered.ID, 1*time.Hour)
		require.NoError(t, err)

		err = repo.SetTTL(ctx, whFailed.ID, 24*time.Hour)
		require.NoError(t, err)

		// Verify different TTLs
		ttlDelivered := GetKeyTTL(t, redisContainer.Addr, "webhook:"+whDelivered.ID)
		ttlFailed := GetKeyTTL(t, redisContainer.Addr, "webhook:"+whFailed.ID)

		assert.Greater(t, ttlDelivered, int64(3500), "Delivered TTL should be ~1 hour (3600s)")
		assert.LessOrEqual(t, ttlDelivered, int64(3600), "Delivered TTL should be <= 1 hour")

		assert.Greater(t, ttlFailed, int64(86300), "Failed TTL should be ~24 hours (86400s)")
		assert.LessOrEqual(t, ttlFailed, int64(86400), "Failed TTL should be <= 24 hours")
	})

	t.Run("delete message ID key", func(t *testing.T) {
		redisContainer, cleanup := SetupRedisContainer(t, ctx)
		defer cleanup()

		repo := CreateTestRepository(t, redisContainer.Addr)
		defer repo.Close(ctx)

		// Create webhook
		wh := webhook.Webhook{
			ID:           "ttl-msgid-test",
			RouteID:      "test-route",
			Payload:      []byte(`{"test": "msgid"}`),
			Headers:      map[string]string{},
			Status:       webhook.Pending,
			RetryCount:   0,
			MaxRetries:   3,
			DeliveryMode: webhook.FIFO,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Consume to create msgid key
		webhooks, err := repo.Consume(ctx, wh.RouteID, webhook.FIFO)
		require.NoError(t, err)
		require.Len(t, webhooks, 1)

		// Verify msgid key exists
		msgIDExists := KeyExists(t, redisContainer.Addr, "webhook:"+wh.ID+":msgid")
		assert.True(t, msgIDExists, "Message ID key should exist after consume")

		// Delete message ID key
		err = repo.DeleteMessageID(ctx, wh.ID)
		require.NoError(t, err)

		// Verify msgid key is deleted
		msgIDExists = KeyExists(t, redisContainer.Addr, "webhook:"+wh.ID+":msgid")
		assert.False(t, msgIDExists, "Message ID key should be deleted")
	})
}
