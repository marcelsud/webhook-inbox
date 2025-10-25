package webhook

import (
	"context"
	"time"
)

/* Small, focused interfaces following "The Go Way"
 * Interfaces abstract behavior, not things
 * Written for users of the API, not just for testing
 */

// Reader provides read operations for webhooks
type Reader interface {
	/* Context is always the first parameter in functions that do I/O
	 * This allows for cancellation, timeouts, and shared values
	 */
	Get(ctx context.Context, id string) (Webhook, error)
	GetByRouteID(ctx context.Context, routeID string, limit int) ([]Webhook, error)
}

// Writer provides write operations for webhooks
type Writer interface {
	/* Store adds a webhook to the appropriate stream (FIFO or PubSub)
	 * Returns the webhook ID and any error
	 */
	Store(ctx context.Context, webhook Webhook) (string, error)
	UpdateStatus(ctx context.Context, id string, status Status) error
	IncrementRetry(ctx context.Context, id string) error
	/* SetTTL sets an expiration time on a webhook
	 * Used to automatically clean up delivered and failed webhooks
	 */
	SetTTL(ctx context.Context, id string, ttl time.Duration) error
	/* DeleteMessageID removes the stream message ID key for a webhook
	 * Used to clean up auxiliary keys when webhooks reach terminal states
	 */
	DeleteMessageID(ctx context.Context, id string) error
}

// StreamConsumer provides operations for consuming webhooks from streams
type StreamConsumer interface {
	/* Consume reads webhooks from the stream for a given route
	 * Blocks until a webhook is available or context is cancelled
	 */
	Consume(ctx context.Context, routeID string, deliveryMode DeliveryMode) ([]Webhook, error)
	/* Acknowledge marks a webhook as successfully processed
	 * This removes it from the pending messages in the consumer group
	 */
	Acknowledge(ctx context.Context, routeID string, deliveryMode DeliveryMode, eventID string) error
}

/* Interface composition - combining small interfaces into larger ones
 * This is preferred over large monolithic interfaces
 */
type Repository interface {
	Reader
	Writer
	StreamConsumer
	Close(ctx context.Context) error
}
