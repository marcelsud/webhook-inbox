package payload

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// eventTypePattern validates event types: hierarchical, full-stop delimited, [a-zA-Z0-9_.]
var eventTypePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+(\.[a-zA-Z0-9_]+)*$`)

// StandardPayload represents a Standard Webhooks compliant payload
type StandardPayload struct {
	// Type is a full-stop delimited type associated with the event
	// Examples: "user.created", "invoice.paid", "order.shipped"
	Type string `json:"type"`

	// Timestamp is the ISO 8601 formatted timestamp of when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Data is the actual event data associated with the event
	Data json.RawMessage `json:"data"`
}

// Validate validates the payload structure according to Standard Webhooks spec
func (p StandardPayload) Validate() error {
	if p.Type == "" {
		return fmt.Errorf("type is required")
	}

	if !eventTypePattern.MatchString(p.Type) {
		return fmt.Errorf("type must be hierarchical and contain only [a-zA-Z0-9_.]: %s", p.Type)
	}

	if p.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	if len(p.Data) == 0 {
		return fmt.Errorf("data is required")
	}

	// Validate that data is valid JSON
	if !json.Valid(p.Data) {
		return fmt.Errorf("data must be valid JSON")
	}

	return nil
}

// MarshalJSON returns the JSON encoding of the payload
func (p StandardPayload) MarshalJSON() ([]byte, error) {
	type Alias StandardPayload
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: p.Timestamp.Format(time.RFC3339Nano),
		Alias:     (*Alias)(&p),
	})
}

// UnmarshalJSON parses the JSON-encoded data and stores the result
func (p *StandardPayload) UnmarshalJSON(data []byte) error {
	type Alias StandardPayload
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("unmarshaling payload: %w", err)
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339Nano, aux.Timestamp)
	if err != nil {
		// Try RFC3339 without nano precision
		timestamp, err = time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return fmt.Errorf("parsing timestamp: %w", err)
		}
	}
	p.Timestamp = timestamp

	return nil
}

// New creates a new StandardPayload with the given type and data
func New(eventType string, data interface{}) (StandardPayload, error) {
	// Marshal data to JSON
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return StandardPayload{}, fmt.Errorf("marshaling data: %w", err)
	}

	payload := StandardPayload{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      dataBytes,
	}

	if err := payload.Validate(); err != nil {
		return StandardPayload{}, fmt.Errorf("validating payload: %w", err)
	}

	return payload, nil
}

// Parse parses a JSON payload into a StandardPayload
func Parse(data []byte) (StandardPayload, error) {
	var payload StandardPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return StandardPayload{}, fmt.Errorf("unmarshaling payload: %w", err)
	}

	if err := payload.Validate(); err != nil {
		return StandardPayload{}, fmt.Errorf("validating payload: %w", err)
	}

	return payload, nil
}

// Bytes returns the JSON-encoded payload as bytes
// The returned bytes are minified (no extra whitespace)
func (p StandardPayload) Bytes() ([]byte, error) {
	return json.Marshal(p)
}

// MatchesEventType checks if the payload's type matches any of the given event types
// Supports exact matching and prefix matching (e.g., "user.*" matches "user.created")
func (p StandardPayload) MatchesEventType(eventTypes []string) bool {
	if len(eventTypes) == 0 {
		// No filter means accept all
		return true
	}

	for _, eventType := range eventTypes {
		// Exact match
		if p.Type == eventType {
			return true
		}

		// Prefix match (e.g., "user.*" matches "user.created", "user.updated")
		if len(eventType) > 2 && eventType[len(eventType)-2:] == ".*" {
			prefix := eventType[:len(eventType)-2]
			if len(p.Type) > len(prefix) && p.Type[:len(prefix)] == prefix && p.Type[len(prefix)] == '.' {
				return true
			}
		}
	}

	return false
}

// ValidateEventType validates an event type format
func ValidateEventType(eventType string) error {
	if eventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}

	// Allow wildcard suffix for filtering
	if len(eventType) > 2 && eventType[len(eventType)-2:] == ".*" {
		eventType = eventType[:len(eventType)-2]
	}

	if !eventTypePattern.MatchString(eventType) {
		return fmt.Errorf("event type must be hierarchical and contain only [a-zA-Z0-9_.]: %s", eventType)
	}

	return nil
}
