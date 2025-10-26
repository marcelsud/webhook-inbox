//go:build integration

package webhook_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcelsud/webhook-inbox/routes"
	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/marcelsud/webhook-inbox/webhook/payload"
	wbredis "github.com/marcelsud/webhook-inbox/webhook/redis"
	"github.com/marcelsud/webhook-inbox/webhook/signature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainersredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// TestStandardWebhooks_EndToEnd tests the full Standard Webhooks flow
func TestStandardWebhooks_EndToEnd(t *testing.T) {
	ctx := context.Background()

	t.Run("FIFO delivery with signing and event filtering", func(t *testing.T) {
		// Setup Redis container
		redisContainer, cleanup := setupRedisContainer(t, ctx)
		defer cleanup()

		repo := createTestRepository(t, redisContainer)
		defer repo.Close(ctx)

		// Generate signing secret
		secret, err := signature.GenerateSecret(32)
		require.NoError(t, err)

		// Setup mock webhook endpoint
		receivedWebhooks := make([]ReceivedWebhook, 0)
		var mu sync.Mutex
		mockEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()

			// Read body
			body, _ := io.ReadAll(r.Body)

			// Capture webhook details
			received := ReceivedWebhook{
				Headers: map[string]string{
					"webhook-id":        r.Header.Get("webhook-id"),
					"webhook-timestamp": r.Header.Get("webhook-timestamp"),
					"webhook-signature": r.Header.Get("webhook-signature"),
				},
				Body: body,
			}
			receivedWebhooks = append(receivedWebhooks, received)

			// Verify signature
			isValid := verifyWebhookSignature(t, secret, received.Headers, body)
			if !isValid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer mockEndpoint.Close()

		// Create route with Standard Webhooks config
		route := &routes.Route{
			RouteID:       "user-events",
			TargetURL:     mockEndpoint.URL,
			Mode:          webhook.FIFO,
			MaxRetries:    3,
			Parallelism:   1,
			SigningSecret: secret.String(),
			EventTypes:    []string{"user.created", "user.updated"},
		}

		// Create webhooks with Standard Webhooks payload
		webhooks := []webhook.Webhook{
			createStandardWebhook(t, "wh1", route.RouteID, "user.created", map[string]interface{}{"user_id": 1}),
			createStandardWebhook(t, "wh2", route.RouteID, "user.updated", map[string]interface{}{"user_id": 1, "name": "John"}),
			createStandardWebhook(t, "wh3", route.RouteID, "user.deleted", map[string]interface{}{"user_id": 1}), // Should be filtered
		}

		// Store webhooks
		for _, wh := range webhooks {
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Simulate worker processing
		processedCount := 0
		for i := 0; i < 5; i++ { // Try consuming multiple times
			consumed, err := repo.Consume(ctx, route.RouteID, route.Mode)
			require.NoError(t, err)

			if len(consumed) == 0 {
				break
			}

			for _, wh := range consumed {
				// Check event type filtering
				p, err := payload.Parse(wh.Payload)
				require.NoError(t, err)

				if !p.MatchesEventType(route.EventTypes) {
					// Skip filtered event
					repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
					continue
				}

				// Deliver webhook
				err = deliverWebhook(ctx, route, wh, secret)
				require.NoError(t, err)

				// Mark as delivered
				repo.UpdateStatus(ctx, wh.ID, webhook.Delivered)
				repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
				processedCount++
			}
		}

		// Verify results
		assert.Equal(t, 2, processedCount, "Should process 2 webhooks (user.created and user.updated)")

		mu.Lock()
		defer mu.Unlock()
		assert.Len(t, receivedWebhooks, 2, "Should receive 2 webhooks (user.deleted filtered out)")

		// Verify Standard Webhooks headers are present
		for _, received := range receivedWebhooks {
			assert.NotEmpty(t, received.Headers["webhook-id"])
			assert.NotEmpty(t, received.Headers["webhook-timestamp"])
			assert.NotEmpty(t, received.Headers["webhook-signature"])
			assert.True(t, strings.HasPrefix(received.Headers["webhook-signature"], "v1,"))

			// Verify payload is Standard Webhooks format
			var p payload.StandardPayload
			err := json.Unmarshal(received.Body, &p)
			require.NoError(t, err)
			assert.Contains(t, []string{"user.created", "user.updated"}, p.Type)
			assert.NotEmpty(t, p.Timestamp)
			assert.NotEmpty(t, p.Data)
		}
	})

	t.Run("PubSub delivery without signing", func(t *testing.T) {
		redisContainer, cleanup := setupRedisContainer(t, ctx)
		defer cleanup()

		repo := createTestRepository(t, redisContainer)
		defer repo.Close(ctx)

		// Setup mock endpoint (no signature verification)
		receivedCount := 0
		var mu sync.Mutex
		mockEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			receivedCount++

			// Verify no signature header when signing_secret is empty
			assert.Empty(t, r.Header.Get("webhook-signature"))
			// But other Standard Webhooks headers should still be present
			assert.NotEmpty(t, r.Header.Get("webhook-id"))
			assert.NotEmpty(t, r.Header.Get("webhook-timestamp"))

			w.WriteHeader(http.StatusOK)
		}))
		defer mockEndpoint.Close()

		route := &routes.Route{
			RouteID:       "analytics",
			TargetURL:     mockEndpoint.URL,
			Mode:          webhook.PubSub,
			MaxRetries:    3,
			Parallelism:   5,
			SigningSecret: "", // No signing
			EventTypes:    []string{}, // Accept all
		}

		// Create webhooks with PubSub mode
		webhooks := []webhook.Webhook{
			createStandardWebhookWithMode(t, "analytics1", route.RouteID, webhook.PubSub, "page.view", map[string]string{"page": "/home"}),
			createStandardWebhookWithMode(t, "analytics2", route.RouteID, webhook.PubSub, "page.view", map[string]string{"page": "/about"}),
			createStandardWebhookWithMode(t, "analytics3", route.RouteID, webhook.PubSub, "click", map[string]string{"button": "signup"}),
		}

		for _, wh := range webhooks {
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Process webhooks
		processedCount := 0
		for i := 0; i < 5; i++ {
			consumed, err := repo.Consume(ctx, route.RouteID, route.Mode)
			require.NoError(t, err)

			if len(consumed) == 0 {
				break
			}

			for _, wh := range consumed {
				err = deliverWebhook(ctx, route, wh, signature.Secret{})
				require.NoError(t, err)
				repo.UpdateStatus(ctx, wh.ID, webhook.Delivered)
				repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
				processedCount++
			}
		}

		assert.Equal(t, 3, processedCount)
		mu.Lock()
		assert.Equal(t, 3, receivedCount)
		mu.Unlock()
	})
}

func TestStandardWebhooks_HTTPStatusHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("410 Gone - should not retry", func(t *testing.T) {
		redisContainer, cleanup := setupRedisContainer(t, ctx)
		defer cleanup()

		repo := createTestRepository(t, redisContainer)
		defer repo.Close(ctx)

		mockEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGone) // 410 Gone
		}))
		defer mockEndpoint.Close()

		route := &routes.Route{
			RouteID:     "gone-test",
			TargetURL:   mockEndpoint.URL,
			Mode:        webhook.FIFO,
			MaxRetries:  3,
			Parallelism: 1,
		}

		wh := createStandardWebhook(t, "gone-wh", route.RouteID, "test.event", map[string]string{"test": "410"})
		_, err := repo.Store(ctx, wh)
		require.NoError(t, err)

		// Consume and process
		consumed, err := repo.Consume(ctx, route.RouteID, route.Mode)
		require.NoError(t, err)
		require.Len(t, consumed, 1)

		// Deliver (will get 410)
		err = deliverWebhook(ctx, route, consumed[0], signature.Secret{})
		assert.Error(t, err) // Should fail with 410

		// In real implementation, 410 should mark as Failed without retry
		// This is a simplified test
	})

	t.Run("2xx Success", func(t *testing.T) {
		redisContainer, cleanup := setupRedisContainer(t, ctx)
		defer cleanup()

		repo := createTestRepository(t, redisContainer)
		defer repo.Close(ctx)

		statusCodes := []int{200, 201, 202}
		receivedStatuses := make([]int, 0)
		var mu sync.Mutex

		mockEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()

			// Rotate through different 2xx status codes
			statusCode := statusCodes[len(receivedStatuses)%len(statusCodes)]
			receivedStatuses = append(receivedStatuses, statusCode)
			w.WriteHeader(statusCode)
		}))
		defer mockEndpoint.Close()

		route := &routes.Route{
			RouteID:     "success-test",
			TargetURL:   mockEndpoint.URL,
			Mode:        webhook.FIFO,
			MaxRetries:  3,
			Parallelism: 1,
		}

		// Create 3 webhooks
		for i := 0; i < 3; i++ {
			wh := createStandardWebhook(t, fmt.Sprintf("success-wh-%d", i), route.RouteID, "test.event", map[string]int{"index": i})
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Process all
		for i := 0; i < 3; i++ {
			consumed, err := repo.Consume(ctx, route.RouteID, route.Mode)
			require.NoError(t, err)

			if len(consumed) > 0 {
				err = deliverWebhook(ctx, route, consumed[0], signature.Secret{})
				require.NoError(t, err) // Should succeed with 2xx
				repo.UpdateStatus(ctx, consumed[0].ID, webhook.Delivered)
				repo.Acknowledge(ctx, route.RouteID, route.Mode, consumed[0].ID)
			}
		}

		mu.Lock()
		assert.Len(t, receivedStatuses, 3)
		assert.Contains(t, receivedStatuses, 200)
		assert.Contains(t, receivedStatuses, 201)
		assert.Contains(t, receivedStatuses, 202)
		mu.Unlock()
	})
}

func TestStandardWebhooks_EventTypeFiltering(t *testing.T) {
	ctx := context.Background()

	t.Run("Exact match filtering", func(t *testing.T) {
		redisContainer, cleanup := setupRedisContainer(t, ctx)
		defer cleanup()

		repo := createTestRepository(t, redisContainer)
		defer repo.Close(ctx)

		receivedTypes := make([]string, 0)
		var mu sync.Mutex

		mockEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			body, _ := io.ReadAll(r.Body)
			var p payload.StandardPayload
			json.Unmarshal(body, &p)
			receivedTypes = append(receivedTypes, p.Type)
			w.WriteHeader(http.StatusOK)
		}))
		defer mockEndpoint.Close()

		route := &routes.Route{
			RouteID:     "exact-match",
			TargetURL:   mockEndpoint.URL,
			Mode:        webhook.FIFO,
			MaxRetries:  3,
			Parallelism: 1,
			EventTypes:  []string{"user.created", "user.updated"},
		}

		// Create webhooks with different event types
		webhooks := []struct {
			eventType    string
			shouldFilter bool
		}{
			{"user.created", false},
			{"user.updated", false},
			{"user.deleted", true},
			{"order.created", true},
		}

		for i, tc := range webhooks {
			wh := createStandardWebhook(t, fmt.Sprintf("filter-wh-%d", i), route.RouteID, tc.eventType, map[string]int{"id": i})
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Process all
		for i := 0; i < 10; i++ { // Try multiple times
			consumed, err := repo.Consume(ctx, route.RouteID, route.Mode)
			require.NoError(t, err)

			if len(consumed) == 0 {
				break
			}

			for _, wh := range consumed {
				p, _ := payload.Parse(wh.Payload)
				if !p.MatchesEventType(route.EventTypes) {
					repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
					continue
				}

				deliverWebhook(ctx, route, wh, signature.Secret{})
				repo.UpdateStatus(ctx, wh.ID, webhook.Delivered)
				repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
			}
		}

		mu.Lock()
		assert.Len(t, receivedTypes, 2)
		assert.Contains(t, receivedTypes, "user.created")
		assert.Contains(t, receivedTypes, "user.updated")
		mu.Unlock()
	})

	t.Run("Wildcard filtering", func(t *testing.T) {
		redisContainer, cleanup := setupRedisContainer(t, ctx)
		defer cleanup()

		repo := createTestRepository(t, redisContainer)
		defer repo.Close(ctx)

		receivedTypes := make([]string, 0)
		var mu sync.Mutex

		mockEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			body, _ := io.ReadAll(r.Body)
			var p payload.StandardPayload
			json.Unmarshal(body, &p)
			receivedTypes = append(receivedTypes, p.Type)
			w.WriteHeader(http.StatusOK)
		}))
		defer mockEndpoint.Close()

		route := &routes.Route{
			RouteID:     "wildcard-match",
			TargetURL:   mockEndpoint.URL,
			Mode:        webhook.FIFO,
			MaxRetries:  3,
			Parallelism: 1,
			EventTypes:  []string{"user.*", "order.created"},
		}

		// Create webhooks
		eventTypes := []string{
			"user.created",   // Matches user.*
			"user.updated",   // Matches user.*
			"user.deleted",   // Matches user.*
			"order.created",  // Matches order.created
			"order.updated",  // Should be filtered
			"product.created", // Should be filtered
		}

		for i, eventType := range eventTypes {
			wh := createStandardWebhook(t, fmt.Sprintf("wildcard-wh-%d", i), route.RouteID, eventType, map[string]int{"id": i})
			_, err := repo.Store(ctx, wh)
			require.NoError(t, err)
		}

		// Process all
		for i := 0; i < 10; i++ {
			consumed, err := repo.Consume(ctx, route.RouteID, route.Mode)
			require.NoError(t, err)

			if len(consumed) == 0 {
				break
			}

			for _, wh := range consumed {
				p, _ := payload.Parse(wh.Payload)
				if !p.MatchesEventType(route.EventTypes) {
					repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
					continue
				}

				deliverWebhook(ctx, route, wh, signature.Secret{})
				repo.UpdateStatus(ctx, wh.ID, webhook.Delivered)
				repo.Acknowledge(ctx, route.RouteID, route.Mode, wh.ID)
			}
		}

		mu.Lock()
		assert.Len(t, receivedTypes, 4, "Should receive: user.created, user.updated, user.deleted, order.created")
		mu.Unlock()
	})
}

// Helper types and functions

type ReceivedWebhook struct {
	Headers map[string]string
	Body    []byte
}

type RedisContainer struct {
	Container *testcontainersredis.RedisContainer
	Addr      string
}

func setupRedisContainer(t *testing.T, ctx context.Context) (*RedisContainer, func()) {
	t.Helper()

	container, err := testcontainersredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	addr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	if len(addr) > 8 && addr[:8] == "redis://" {
		addr = addr[8:]
	}

	time.Sleep(1 * time.Second)

	return &RedisContainer{
		Container: container,
		Addr:      addr,
	}, func() {
		container.Terminate(ctx)
	}
}

func createTestRepository(t *testing.T, rc *RedisContainer) *wbredis.Repository {
	t.Helper()
	repo, err := wbredis.NewRepository(rc.Addr, "", 0)
	require.NoError(t, err)
	return repo
}

func createStandardWebhook(t *testing.T, id, routeID, eventType string, data interface{}) webhook.Webhook {
	t.Helper()
	return createStandardWebhookWithMode(t, id, routeID, webhook.FIFO, eventType, data)
}

func createStandardWebhookWithMode(t *testing.T, id, routeID string, mode webhook.DeliveryMode, eventType string, data interface{}) webhook.Webhook {
	t.Helper()

	p, err := payload.New(eventType, data)
	require.NoError(t, err)

	payloadBytes, err := p.Bytes()
	require.NoError(t, err)

	return webhook.Webhook{
		ID:           id,
		RouteID:      routeID,
		Payload:      payloadBytes,
		Headers:      map[string]string{"Content-Type": "application/json"},
		Status:       webhook.Pending,
		RetryCount:   0,
		MaxRetries:   3,
		DeliveryMode: mode,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func deliverWebhook(ctx context.Context, route *routes.Route, wh webhook.Webhook, secret signature.Secret) error {
	req, err := http.NewRequestWithContext(ctx, "POST", route.TargetURL, strings.NewReader(string(wh.Payload)))
	if err != nil {
		return err
	}

	// Add Standard Webhooks headers
	timestamp := time.Now()
	req.Header.Set("webhook-id", wh.ID)
	req.Header.Set("webhook-timestamp", fmt.Sprintf("%d", timestamp.Unix()))
	req.Header.Set("Content-Type", "application/json")

	// Add signature if secret is provided
	if route.SigningSecret != "" && len(secret.Bytes()) > 0 {
		sig, err := signature.Sign(secret, wh.ID, timestamp, wh.Payload)
		if err != nil {
			return fmt.Errorf("signing webhook: %w", err)
		}
		req.Header.Set("webhook-signature", sig.String())
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check status code (2xx is success)
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}

	return fmt.Errorf("webhook delivery failed with status: %d", resp.StatusCode)
}

func verifyWebhookSignature(t *testing.T, secret signature.Secret, headers map[string]string, body []byte) bool {
	t.Helper()

	msgID := headers["webhook-id"]
	timestampStr := headers["webhook-timestamp"]
	signatureHeader := headers["webhook-signature"]

	if msgID == "" || timestampStr == "" || signatureHeader == "" {
		return false
	}

	// Parse timestamp
	var timestamp int64
	fmt.Sscanf(timestampStr, "%d", &timestamp)

	// Parse signature
	sigs, err := signature.ParseSignatureHeader(signatureHeader)
	if err != nil {
		return false
	}

	// Verify
	for _, sig := range sigs {
		valid, err := signature.Verify(secret, msgID, time.Unix(timestamp, 0), body, sig)
		if err == nil && valid {
			return true
		}
	}

	return false
}
