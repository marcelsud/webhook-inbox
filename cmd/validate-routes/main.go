package main

import (
	"fmt"
	"os"

	"github.com/marcelsud/webhook-inbox/routes"
)

/* validate-routes - Standalone CLI tool to validate routes.yaml
 * Usage: go run cmd/validate-routes/main.go [routes.yaml]
 * Exit codes: 0 = valid, 1 = invalid
 */

func main() {
	// Get routes file path from args or use default
	routesFile := "routes.yaml"
	if len(os.Args) > 1 {
		routesFile = os.Args[1]
	}

	// Print validation header
	fmt.Printf("Validating routes file: %s\n", routesFile)
	fmt.Println(string(make([]byte, 50))) // separator line

	// Create loader and attempt to load routes
	loader := routes.NewLoader()
	if err := loader.Load(routesFile); err != nil {
		fmt.Fprintf(os.Stderr, "❌ VALIDATION FAILED\n\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Success - print loaded routes
	loadedRoutes := loader.List()
	fmt.Printf("✓ VALIDATION PASSED\n\n")
	fmt.Printf("Loaded %d route(s):\n", len(loadedRoutes))

	for i, route := range loadedRoutes {
		fmt.Printf("\n%d. Route: %s\n", i+1, route.RouteID)
		fmt.Printf("   Target URL:    %s\n", route.TargetURL)
		fmt.Printf("   Mode:          %s\n", route.Mode)
		fmt.Printf("   Parallelism:   %d\n", route.Parallelism)
		fmt.Printf("   Max Retries:   %d\n", route.MaxRetries)
		fmt.Printf("   Retry Backoff: %s\n", route.RetryBackoff)
		fmt.Printf("   Expected Status: %d\n", route.ExpectedStatus)

		if route.DeliveredTTLHours != nil {
			fmt.Printf("   Delivered TTL: %d hours\n", *route.DeliveredTTLHours)
		}
		if route.FailedTTLHours != nil {
			fmt.Printf("   Failed TTL:    %d hours\n", *route.FailedTTLHours)
		}
	}

	fmt.Printf("\n✓ All routes are valid!\n")
	os.Exit(0)
}
