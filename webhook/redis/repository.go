package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/redis/go-redis/v9"
)

/* Redis Streams implementation of webhook.Repository
 * Uses Redis Streams for message queueing with consumer groups
 * Uses Redis Hashes for webhook metadata storage
 */

const (
	streamPrefix        = "webhooks"        // Stream naming: webhooks:fifo:{route_id} or webhooks:pubsub:{route_id}
	hashPrefix          = "webhook"         // Hash naming: webhook:{webhook_id}
	consumerGroupPrefix = "webhook-workers" // Consumer group naming: webhook-workers-{route_id}
	consumerName        = "worker"          // Consumer name (can be made dynamic for multiple workers)
)

type Repository struct {
	client *redis.Client
}

// NewRepository creates a new Redis repository
func NewRepository(addr, password string, db int) (*Repository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to Redis: %w", err)
	}

	return &Repository{
		client: client,
	}, nil
}

// Store adds a webhook to the appropriate Redis Stream
func (r *Repository) Store(ctx context.Context, wh webhook.Webhook) (string, error) {
	// Store webhook metadata in hash for quick lookups
	hashKey := fmt.Sprintf("%s:%s", hashPrefix, wh.ID)

	headersJSON, err := json.Marshal(wh.Headers)
	if err != nil {
		return "", fmt.Errorf("marshaling headers: %w", err)
	}

	err = r.client.HSet(ctx, hashKey, map[string]interface{}{
		"id":            wh.ID,
		"route_id":      wh.RouteID,
		"payload":       wh.Payload,
		"headers":       string(headersJSON),
		"status":        wh.Status.String(),
		"retry_count":   wh.RetryCount,
		"max_retries":   wh.MaxRetries,
		"delivery_mode": wh.DeliveryMode.String(),
		"created_at":    wh.CreatedAt.Unix(),
		"updated_at":    wh.UpdatedAt.Unix(),
	}).Err()
	if err != nil {
		return "", fmt.Errorf("storing webhook metadata: %w", err)
	}

	// Add to stream
	streamKey := getStreamKey(wh.RouteID, wh.DeliveryMode)

	// Create consumer group if it doesn't exist
	groupName := fmt.Sprintf("%s-%s", consumerGroupPrefix, wh.RouteID)
	r.client.XGroupCreateMkStream(ctx, streamKey, groupName, "0")
	// Ignore error if group already exists

	// Add webhook to stream
	streamData := map[string]interface{}{
		"event_id": wh.ID,
		"route_id": wh.RouteID,
		"payload":  wh.Payload,
		"headers":  string(headersJSON),
	}

	_, err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: streamData,
	}).Result()
	if err != nil {
		return "", fmt.Errorf("adding to stream: %w", err)
	}

	return wh.ID, nil
}

// Get retrieves a webhook by ID from Redis hash
func (r *Repository) Get(ctx context.Context, id string) (webhook.Webhook, error) {
	hashKey := fmt.Sprintf("%s:%s", hashPrefix, id)

	data, err := r.client.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return webhook.Webhook{}, fmt.Errorf("getting webhook: %w", err)
	}
	if len(data) == 0 {
		return webhook.Webhook{}, fmt.Errorf("webhook not found: %s", id)
	}

	// Parse headers
	headers := make(map[string]string)
	if headersStr, ok := data["headers"]; ok && headersStr != "" {
		if err := json.Unmarshal([]byte(headersStr), &headers); err != nil {
			return webhook.Webhook{}, fmt.Errorf("unmarshaling headers: %w", err)
		}
	}

	// Parse timestamps
	createdAt := time.Unix(parseInt64(data["created_at"]), 0)
	updatedAt := time.Unix(parseInt64(data["updated_at"]), 0)

	wh := webhook.Webhook{
		ID:           data["id"],
		RouteID:      data["route_id"],
		Payload:      []byte(data["payload"]),
		Headers:      headers,
		Status:       webhook.NewStatus(data["status"]),
		RetryCount:   int(parseInt64(data["retry_count"])),
		MaxRetries:   int(parseInt64(data["max_retries"])),
		DeliveryMode: webhook.NewDeliveryMode(data["delivery_mode"]),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	return wh, nil
}

// GetByRouteID retrieves webhooks for a specific route (limited for listing)
func (r *Repository) GetByRouteID(ctx context.Context, routeID string, limit int) ([]webhook.Webhook, error) {
	// This is a simplified implementation
	// In production, you might want to use a secondary index or scan pattern
	return nil, fmt.Errorf("GetByRouteID not implemented yet")
}

// UpdateStatus updates the status of a webhook
func (r *Repository) UpdateStatus(ctx context.Context, id string, status webhook.Status) error {
	hashKey := fmt.Sprintf("%s:%s", hashPrefix, id)

	err := r.client.HSet(ctx, hashKey, map[string]interface{}{
		"status":     status.String(),
		"updated_at": time.Now().Unix(),
	}).Err()
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	return nil
}

// IncrementRetry increments the retry count for a webhook
func (r *Repository) IncrementRetry(ctx context.Context, id string) error {
	hashKey := fmt.Sprintf("%s:%s", hashPrefix, id)

	err := r.client.HIncrBy(ctx, hashKey, "retry_count", 1).Err()
	if err != nil {
		return fmt.Errorf("incrementing retry count: %w", err)
	}

	err = r.client.HSet(ctx, hashKey, "updated_at", time.Now().Unix()).Err()
	if err != nil {
		return fmt.Errorf("updating timestamp: %w", err)
	}

	return nil
}

// Consume reads webhooks from a stream for a given route
func (r *Repository) Consume(ctx context.Context, routeID string, deliveryMode webhook.DeliveryMode) ([]webhook.Webhook, error) {
	streamKey := getStreamKey(routeID, deliveryMode)
	groupName := fmt.Sprintf("%s-%s", consumerGroupPrefix, routeID)

	// Create consumer group if it doesn't exist
	r.client.XGroupCreateMkStream(ctx, streamKey, groupName, "0")
	// Ignore error if group already exists

	// Read from stream using consumer group
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    groupName,
		Consumer: consumerName,
		Streams:  []string{streamKey, ">"},
		Count:    1,
		Block:    1 * time.Second, // Shorter timeout for better responsiveness
	}).Result()
	if err == redis.Nil {
		// No messages available
		return []webhook.Webhook{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading from stream: %w", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return []webhook.Webhook{}, nil
	}

	var webhooks []webhook.Webhook
	for _, msg := range streams[0].Messages {
		eventID, ok := msg.Values["event_id"].(string)
		if !ok {
			continue
		}

		// Retrieve full webhook data from hash
		wh, err := r.Get(ctx, eventID)
		if err != nil {
			continue
		}

		// Store the stream message ID in the webhook for acknowledgment
		// We'll store it in a separate hash field
		msgIDKey := fmt.Sprintf("%s:%s:msgid", hashPrefix, eventID)
		r.client.Set(ctx, msgIDKey, msg.ID, 24*time.Hour) // TTL of 24 hours

		webhooks = append(webhooks, wh)
	}

	return webhooks, nil
}

// Acknowledge marks a webhook as successfully processed
func (r *Repository) Acknowledge(ctx context.Context, routeID string, deliveryMode webhook.DeliveryMode, eventID string) error {
	streamKey := getStreamKey(routeID, deliveryMode)
	groupName := fmt.Sprintf("%s-%s", consumerGroupPrefix, routeID)

	// Get the stream message ID for this webhook
	msgIDKey := fmt.Sprintf("%s:%s:msgid", hashPrefix, eventID)
	msgID, err := r.client.Get(ctx, msgIDKey).Result()
	if err == redis.Nil {
		// Message ID not found, might have been already acknowledged or expired
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting message ID: %w", err)
	}

	// Acknowledge the message in the stream
	err = r.client.XAck(ctx, streamKey, groupName, msgID).Err()
	if err != nil {
		return fmt.Errorf("acknowledging message: %w", err)
	}

	// Clean up the message ID key
	r.client.Del(ctx, msgIDKey)

	return nil
}

// SetTTL sets an expiration time on a webhook hash
func (r *Repository) SetTTL(ctx context.Context, id string, ttl time.Duration) error {
	hashKey := fmt.Sprintf("%s:%s", hashPrefix, id)

	err := r.client.Expire(ctx, hashKey, ttl).Err()
	if err != nil {
		return fmt.Errorf("setting TTL on webhook: %w", err)
	}

	return nil
}

// DeleteMessageID removes the message ID key for a webhook
func (r *Repository) DeleteMessageID(ctx context.Context, id string) error {
	msgIDKey := fmt.Sprintf("%s:%s:msgid", hashPrefix, id)
	return r.client.Del(ctx, msgIDKey).Err()
}

// Close closes the Redis connection
func (r *Repository) Close(ctx context.Context) error {
	return r.client.Close()
}

// GetClient returns the underlying Redis client for advanced operations
func (r *Repository) GetClient() *redis.Client {
	return r.client
}

// Helper functions

func getStreamKey(routeID string, mode webhook.DeliveryMode) string {
	return fmt.Sprintf("%s:%s:%s", streamPrefix, mode.String(), routeID)
}

func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}
