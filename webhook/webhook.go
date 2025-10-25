package webhook

import "time"

/* Webhook represents a received webhook message in the system
 * Uses value semantics as it represents data, not behavior
 */
type Webhook struct {
	ID           string
	RouteID      string
	Payload      []byte
	Headers      map[string]string
	Status       Status
	RetryCount   int
	MaxRetries   int
	NextRetryAt  time.Time
	DeliveryMode DeliveryMode
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
