package routes

import (
	"fmt"
	"os"

	"github.com/marcelsud/webhook-inbox/webhook"
	"gopkg.in/yaml.v3"
)

/* Loader manages route configuration from routes.yaml
 * Provides in-memory lookup for fast access
 */

// Config represents the structure of routes.yaml
type Config struct {
	Routes []RouteConfig `yaml:"routes"`
}

// RouteConfig represents a single route in the YAML file
type RouteConfig struct {
	RouteID           string `yaml:"route_id"`
	TargetURL         string `yaml:"target_url"`
	Mode              string `yaml:"mode"`
	MaxRetries        int    `yaml:"max_retries"`
	RetryBackoff      string `yaml:"retry_backoff"`
	Parallelism       int    `yaml:"parallelism"`
	ExpectedStatus    int    `yaml:"expected_status"`     // Default: 202
	DeliveredTTLHours *int   `yaml:"delivered_ttl_hours"` // Optional: override global default
	FailedTTLHours    *int   `yaml:"failed_ttl_hours"`    // Optional: override global default
}

// Loader holds the loaded routes
type Loader struct {
	routes map[string]*Route
}

// NewLoader creates a new route loader
func NewLoader() *Loader {
	return &Loader{
		routes: make(map[string]*Route),
	}
}

// Load reads and parses the routes.yaml file
func (l *Loader) Load(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading routes file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing routes YAML: %w", err)
	}

	// Convert and validate routes
	for _, rc := range config.Routes {
		// Set default expected status to 202 if not specified
		expectedStatus := rc.ExpectedStatus
		if expectedStatus == 0 {
			expectedStatus = 202
		}

		route := &Route{
			RouteID:           rc.RouteID,
			TargetURL:         rc.TargetURL,
			Mode:              webhook.NewDeliveryMode(rc.Mode),
			MaxRetries:        rc.MaxRetries,
			RetryBackoff:      rc.RetryBackoff,
			Parallelism:       rc.Parallelism,
			ExpectedStatus:    expectedStatus,
			DeliveredTTLHours: rc.DeliveredTTLHours,
			FailedTTLHours:    rc.FailedTTLHours,
		}

		if err := route.Validate(); err != nil {
			return fmt.Errorf("validating route: %w", err)
		}

		l.routes[route.RouteID] = route
	}

	return nil
}

// Get retrieves a route by its ID
func (l *Loader) Get(routeID string) (*Route, error) {
	route, exists := l.routes[routeID]
	if !exists {
		return nil, fmt.Errorf("route not found: %s", routeID)
	}
	return route, nil
}

// List returns all loaded routes
func (l *Loader) List() []*Route {
	routes := make([]*Route, 0, len(l.routes))
	for _, route := range l.routes {
		routes = append(routes, route)
	}
	return routes
}

// Exists checks if a route ID exists
func (l *Loader) Exists(routeID string) bool {
	_, exists := l.routes[routeID]
	return exists
}
