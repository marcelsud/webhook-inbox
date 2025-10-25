package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

/* Service represents the business logic layer
 * Uses pointer semantics as it's an API, not data
 */

// UseCase defines the business operations for webhook management
type UseCase interface {
	Receive(ctx context.Context, routeID string, deliveryMode DeliveryMode, payload []byte, headers map[string]string, maxRetries int) (string, error)
	UpdateStatus(ctx context.Context, id string, status Status) error
	IncrementRetry(ctx context.Context, id string) error
}

type Service struct {
	Repo Repository
}

// NewService creates a new webhook service with dependency injection
func NewService(repo Repository) *Service {
	return &Service{
		Repo: repo,
	}
}

// Receive accepts a new webhook and stores it in the appropriate stream
func (s *Service) Receive(ctx context.Context, routeID string, deliveryMode DeliveryMode, payload []byte, headers map[string]string, maxRetries int) (string, error) {
	if err := deliveryMode.Validate(); err != nil {
		return "", fmt.Errorf("validating delivery mode: %w", err)
	}

	webhook := Webhook{
		ID:           uuid.New().String(),
		RouteID:      routeID,
		Payload:      payload,
		Headers:      headers,
		Status:       Pending,
		RetryCount:   0,
		MaxRetries:   maxRetries,
		DeliveryMode: deliveryMode,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	id, err := s.Repo.Store(ctx, webhook)
	if err != nil {
		return "", fmt.Errorf("storing webhook: %w", err)
	}

	return id, nil
}

// UpdateStatus updates the status of a webhook
func (s *Service) UpdateStatus(ctx context.Context, id string, status Status) error {
	if err := status.Validate(); err != nil {
		return fmt.Errorf("validating status: %w", err)
	}

	err := s.Repo.UpdateStatus(ctx, id, status)
	if err != nil {
		return fmt.Errorf("updating webhook status: %w", err)
	}
	return nil
}

// IncrementRetry increments the retry count for a webhook
func (s *Service) IncrementRetry(ctx context.Context, id string) error {
	err := s.Repo.IncrementRetry(ctx, id)
	if err != nil {
		return fmt.Errorf("incrementing retry count: %w", err)
	}
	return nil
}
