package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/marcelsud/webhook-inbox/config"
	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/marcelsud/webhook-inbox/webhook/payload"
	"github.com/marcelsud/webhook-inbox/webhook/signature"
)

/* Route represents a webhook destination configuration
 * Maps route_id to target URL with delivery settings
 */
type Route struct {
	RouteID           string
	TargetURL         string
	Mode              webhook.DeliveryMode
	MaxRetries        int
	RetryBackoff      string   // Expression like "pow(2, retried) * 1000"
	Parallelism       int      // 1 for FIFO, >1 for PubSub
	ExpectedStatus    int      // Expected HTTP status code: 200, 201, or 202 (default: 202)
	DeliveredTTLHours *int     // Optional: TTL for delivered webhooks in hours
	FailedTTLHours    *int     // Optional: TTL for failed webhooks in hours
	SigningSecret     string   // Standard Webhooks signing secret (whsec_ prefix)
	EventTypes        []string // Event types to filter (e.g., ["user.created", "user.*"])
}

// Validate checks if the route configuration is valid
func (r *Route) Validate() error {
	if r.RouteID == "" {
		return fmt.Errorf("route_id cannot be empty")
	}
	if r.TargetURL == "" {
		return fmt.Errorf("target_url cannot be empty for route %s", r.RouteID)
	}
	if err := r.Mode.Validate(); err != nil {
		return fmt.Errorf("invalid mode for route %s: %w", r.RouteID, err)
	}
	if r.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative for route %s", r.RouteID)
	}
	if r.Parallelism < 1 {
		return fmt.Errorf("parallelism must be at least 1 for route %s", r.RouteID)
	}
	// FIFO mode should have parallelism=1 for ordering guarantees
	if r.Mode == webhook.FIFO && r.Parallelism > 1 {
		return fmt.Errorf("FIFO mode requires parallelism=1 for route %s (got %d)", r.RouteID, r.Parallelism)
	}
	// Validate expected status code (only 200, 201, 202 allowed)
	if r.ExpectedStatus != 200 && r.ExpectedStatus != 201 && r.ExpectedStatus != 202 {
		return fmt.Errorf("expected_status must be 200, 201, or 202 for route %s (got %d)", r.RouteID, r.ExpectedStatus)
	}
	// Validate TTL values if provided
	if r.DeliveredTTLHours != nil && *r.DeliveredTTLHours < 0 {
		return fmt.Errorf("delivered_ttl_hours cannot be negative for route %s", r.RouteID)
	}
	if r.FailedTTLHours != nil && *r.FailedTTLHours < 0 {
		return fmt.Errorf("failed_ttl_hours cannot be negative for route %s", r.RouteID)
	}
	// Validate signing secret if provided (Standard Webhooks)
	if r.SigningSecret != "" {
		if !strings.HasPrefix(r.SigningSecret, signature.SecretPrefix) {
			return fmt.Errorf("signing_secret must start with %s for route %s", signature.SecretPrefix, r.RouteID)
		}
		// Try to parse the secret to validate format
		if _, err := signature.ParseSecret(r.SigningSecret); err != nil {
			return fmt.Errorf("invalid signing_secret for route %s: %w", r.RouteID, err)
		}
	}
	// Validate event types if provided
	for _, eventType := range r.EventTypes {
		if err := payload.ValidateEventType(eventType); err != nil {
			return fmt.Errorf("invalid event_type '%s' for route %s: %w", eventType, r.RouteID, err)
		}
	}
	return nil
}

// GetDeliveredTTL returns the TTL for delivered webhooks
// Priority: route-specific > config > default (1 hour)
func (r *Route) GetDeliveredTTL(cfg *config.Config) time.Duration {
	hours := 1 // default
	if cfg != nil {
		hours = cfg.GetWebhookDeliveredTTLHours()
	}
	if r.DeliveredTTLHours != nil {
		hours = *r.DeliveredTTLHours
	}
	return time.Duration(hours) * time.Hour
}

// GetFailedTTL returns the TTL for failed webhooks
// Priority: route-specific > config > default (24 hours)
func (r *Route) GetFailedTTL(cfg *config.Config) time.Duration {
	hours := 24 // default
	if cfg != nil {
		hours = cfg.GetWebhookFailedTTLHours()
	}
	if r.FailedTTLHours != nil {
		hours = *r.FailedTTLHours
	}
	return time.Duration(hours) * time.Hour
}
