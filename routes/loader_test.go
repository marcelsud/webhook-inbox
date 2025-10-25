package routes_test

import (
	"os"
	"testing"

	"github.com/marcelsud/webhook-inbox/routes"
	"github.com/marcelsud/webhook-inbox/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load(t *testing.T) {
	t.Run("success - valid routes file", func(t *testing.T) {
		// Create temporary routes file
		content := `
routes:
  - route_id: "test-fifo"
    target_url: "https://example.com/webhook"
    mode: "fifo"
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 1
  - route_id: "test-pubsub"
    target_url: "https://example.com/pubsub"
    mode: "pubsub"
    max_retries: 5
    retry_backoff: "1000"
    parallelism: 10
`
		tmpFile, err := os.CreateTemp("", "routes-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		tmpFile.Close()

		// Load routes
		loader := routes.NewLoader()
		err = loader.Load(tmpFile.Name())

		require.NoError(t, err)

		// Verify routes loaded correctly
		allRoutes := loader.List()
		assert.Len(t, allRoutes, 2)

		// Check FIFO route
		route, err := loader.Get("test-fifo")
		require.NoError(t, err)
		assert.Equal(t, "test-fifo", route.RouteID)
		assert.Equal(t, "https://example.com/webhook", route.TargetURL)
		assert.Equal(t, webhook.FIFO, route.Mode)
		assert.Equal(t, 3, route.MaxRetries)
		assert.Equal(t, 1, route.Parallelism)

		// Check PubSub route
		route, err = loader.Get("test-pubsub")
		require.NoError(t, err)
		assert.Equal(t, webhook.PubSub, route.Mode)
		assert.Equal(t, 10, route.Parallelism)
	})

	t.Run("error - file not found", func(t *testing.T) {
		loader := routes.NewLoader()
		err := loader.Load("nonexistent.yaml")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading routes file")
	})

	t.Run("error - invalid YAML", func(t *testing.T) {
		content := `invalid yaml content: [[[`

		tmpFile, err := os.CreateTemp("", "routes-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		tmpFile.Close()

		loader := routes.NewLoader()
		err = loader.Load(tmpFile.Name())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing routes YAML")
	})

	t.Run("error - FIFO with parallelism > 1", func(t *testing.T) {
		content := `
routes:
  - route_id: "invalid-fifo"
    target_url: "https://example.com"
    mode: "fifo"
    max_retries: 3
    retry_backoff: "1000"
    parallelism: 5
`
		tmpFile, err := os.CreateTemp("", "routes-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		tmpFile.Close()

		loader := routes.NewLoader()
		err = loader.Load(tmpFile.Name())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "FIFO mode requires parallelism=1")
	})
}

func TestLoader_Get(t *testing.T) {
	t.Run("route not found", func(t *testing.T) {
		loader := routes.NewLoader()

		_, err := loader.Get("nonexistent")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "route not found")
	})
}

func TestLoader_Exists(t *testing.T) {
	content := `
routes:
  - route_id: "test-route"
    target_url: "https://example.com"
    mode: "fifo"
    max_retries: 3
    retry_backoff: "1000"
    parallelism: 1
`
	tmpFile, err := os.CreateTemp("", "routes-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	loader := routes.NewLoader()
	err = loader.Load(tmpFile.Name())
	require.NoError(t, err)

	t.Run("route exists", func(t *testing.T) {
		exists := loader.Exists("test-route")
		assert.True(t, exists)
	})

	t.Run("route does not exist", func(t *testing.T) {
		exists := loader.Exists("nonexistent")
		assert.False(t, exists)
	})
}

func TestRoute_Validate(t *testing.T) {
	t.Run("valid FIFO route", func(t *testing.T) {
		route := &routes.Route{
			RouteID:        "test",
			TargetURL:      "https://example.com",
			Mode:           webhook.FIFO,
			MaxRetries:     3,
			RetryBackoff:   "1000",
			Parallelism:    1,
			ExpectedStatus: 200,
		}

		err := route.Validate()
		require.NoError(t, err)
	})

	t.Run("valid PubSub route", func(t *testing.T) {
		route := &routes.Route{
			RouteID:        "test",
			TargetURL:      "https://example.com",
			Mode:           webhook.PubSub,
			MaxRetries:     3,
			RetryBackoff:   "1000",
			Parallelism:    10,
			ExpectedStatus: 200,
		}

		err := route.Validate()
		require.NoError(t, err)
	})

	t.Run("error - empty route_id", func(t *testing.T) {
		route := &routes.Route{
			RouteID:     "",
			TargetURL:   "https://example.com",
			Mode:        webhook.FIFO,
			Parallelism: 1,
		}

		err := route.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "route_id cannot be empty")
	})

	t.Run("error - empty target_url", func(t *testing.T) {
		route := &routes.Route{
			RouteID:     "test",
			TargetURL:   "",
			Mode:        webhook.FIFO,
			Parallelism: 1,
		}

		err := route.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target_url cannot be empty")
	})

	t.Run("error - negative max_retries", func(t *testing.T) {
		route := &routes.Route{
			RouteID:     "test",
			TargetURL:   "https://example.com",
			Mode:        webhook.FIFO,
			MaxRetries:  -1,
			Parallelism: 1,
		}

		err := route.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_retries cannot be negative")
	})

	t.Run("error - parallelism less than 1", func(t *testing.T) {
		route := &routes.Route{
			RouteID:     "test",
			TargetURL:   "https://example.com",
			Mode:        webhook.FIFO,
			Parallelism: 0,
		}

		err := route.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parallelism must be at least 1")
	})
}
