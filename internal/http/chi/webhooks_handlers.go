package chi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/marcelsud/webhook-inbox/routes"
	"github.com/marcelsud/webhook-inbox/webhook"
)

// WebhookHandlers sets up the webhook API routes
func WebhookHandlers(ctx context.Context, webhookService webhook.UseCase, routeLoader *routes.Loader) *chi.Mux {
	logger := httplog.NewLogger("webhook-api", httplog.Options{
		JSON: true,
	})

	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Webhook API routes
	r.Route("/v1", func(r chi.Router) {
		// List available routes
		r.Get("/routes", getRoutes(routeLoader).ServeHTTP)

		// Send event to route
		r.Post("/routes/{route_id}/events", postWebhook(webhookService, routeLoader).ServeHTTP)
	})

	return r
}
