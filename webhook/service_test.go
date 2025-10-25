package webhook_test

import (
	"context"
	"testing"

	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/marcelsud/webhook-inbox/webhook/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceive(t *testing.T) {
	ctx := context.Background()

	t.Run("success - FIFO mode", func(t *testing.T) {
		repo := mocks.NewRepository(t)
		service := webhook.NewService(repo)

		payload := []byte(`{"test": "data"}`)
		headers := map[string]string{"Content-Type": "application/json"}

		// Mock expects Store to be called and returns webhook ID
		repo.On("Store", ctx, webhook.MatchWebhook(func(wh webhook.Webhook) bool {
			return wh.RouteID == "test-route" &&
				wh.DeliveryMode == webhook.FIFO &&
				string(wh.Payload) == string(payload) &&
				wh.Status == webhook.Pending &&
				wh.RetryCount == 0 &&
				wh.MaxRetries == 3
		})).Return("webhook-123", nil)

		id, err := service.Receive(ctx, "test-route", webhook.FIFO, payload, headers, 3)

		require.NoError(t, err)
		assert.Equal(t, "webhook-123", id)
		repo.AssertExpectations(t)
	})

	t.Run("success - PubSub mode", func(t *testing.T) {
		repo := mocks.NewRepository(t)
		service := webhook.NewService(repo)

		payload := []byte(`{"event": "user.created"}`)
		headers := map[string]string{"X-Event-Type": "user.created"}

		repo.On("Store", ctx, webhook.MatchWebhook(func(wh webhook.Webhook) bool {
			return wh.DeliveryMode == webhook.PubSub
		})).Return("webhook-456", nil)

		id, err := service.Receive(ctx, "analytics", webhook.PubSub, payload, headers, 5)

		require.NoError(t, err)
		assert.Equal(t, "webhook-456", id)
		repo.AssertExpectations(t)
	})

	t.Run("invalid delivery mode", func(t *testing.T) {
		repo := mocks.NewRepository(t)
		service := webhook.NewService(repo)

		invalidMode := webhook.DeliveryMode(999)

		_, err := service.Receive(ctx, "test", invalidMode, []byte("test"), nil, 3)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validating delivery mode")
	})
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewRepository(t)
		service := webhook.NewService(repo)

		repo.On("UpdateStatus", ctx, "webhook-123", webhook.Delivered).Return(nil)

		err := service.UpdateStatus(ctx, "webhook-123", webhook.Delivered)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("invalid status", func(t *testing.T) {
		repo := mocks.NewRepository(t)
		service := webhook.NewService(repo)

		invalidStatus := webhook.Status(999)

		err := service.UpdateStatus(ctx, "webhook-123", invalidStatus)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validating status")
	})
}

func TestIncrementRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewRepository(t)
		service := webhook.NewService(repo)

		repo.On("IncrementRetry", ctx, "webhook-123").Return(nil)

		err := service.IncrementRetry(ctx, "webhook-123")

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}
