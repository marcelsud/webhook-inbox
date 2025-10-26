package chi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/marcelsud/webhook-inbox/routes"
	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/marcelsud/webhook-inbox/webhook/payload"
)

/* HTTP layer DTOs for webhook API
 * Separate from domain entities to avoid leaking internal structure
 */

// webhookRequest represents the incoming webhook payload (raw)
type webhookRequest struct {
	Body    []byte
	Headers map[string]string
}

// webhookResponse represents the API response when creating a webhook
type webhookResponse struct {
	EventID string `json:"event_id"`
	RouteID string `json:"route_id"`
}

// routeResponse represents a route in the API
type routeResponse struct {
	RouteID        string `json:"route_id"`
	TargetURL      string `json:"target_url"`
	Mode           string `json:"mode"`
	MaxRetries     int    `json:"max_retries"`
	RetryBackoff   string `json:"retry_backoff"`
	Parallelism    int    `json:"parallelism"`
	ExpectedStatus int    `json:"expected_status"`
}

// postWebhook handles POST /v1/routes/:route_id/events
func postWebhook(webhookService webhook.UseCase, routeLoader *routes.Loader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeID := chi.URLParam(r, "route_id")
		if routeID == "" {
			http.Error(w, "route_id is required", http.StatusBadRequest)
			return
		}

		// Check if route exists
		route, err := routeLoader.Get(routeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("route not found: %s", routeID), http.StatusNotFound)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Validate Standard Webhooks payload format
		if _, err := payload.Parse(body); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload format: %v (expected Standard Webhooks format with type, timestamp, and data)", err), http.StatusBadRequest)
			return
		}

		// Extract headers (optionally filter to only forward certain headers)
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		// Create webhook
		eventID, err := webhookService.Receive(
			r.Context(),
			routeID,
			route.Mode,
			body,
			headers,
			route.MaxRetries,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return 202 Accepted with event ID
		w.WriteHeader(http.StatusAccepted)
		response := webhookResponse{
			EventID: eventID,
			RouteID: routeID,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// getRoutes handles GET /v1/routes
func getRoutes(routeLoader *routes.Loader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allRoutes := routeLoader.List()

		responses := make([]routeResponse, 0, len(allRoutes))
		for _, route := range allRoutes {
			responses = append(responses, routeResponse{
				RouteID:        route.RouteID,
				TargetURL:      route.TargetURL,
				Mode:           route.Mode.String(),
				MaxRetries:     route.MaxRetries,
				RetryBackoff:   route.RetryBackoff,
				Parallelism:    route.Parallelism,
				ExpectedStatus: route.ExpectedStatus,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responses); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
