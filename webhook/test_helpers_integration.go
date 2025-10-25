//go:build integration

package webhook

import (
	"fmt"
	"testing"
	"time"
)

// GenerateID generates a unique webhook ID for testing
func GenerateID(t *testing.T, index int) string {
	t.Helper()
	return fmt.Sprintf("test-webhook-%d-%d", index, time.Now().UnixNano())
}
