package payload

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("success - creates valid payload", func(t *testing.T) {
		data := map[string]interface{}{
			"user_id": 123,
			"action":  "created",
		}

		payload, err := New("user.created", data)
		require.NoError(t, err)
		assert.Equal(t, "user.created", payload.Type)
		assert.False(t, payload.Timestamp.IsZero())
		assert.NotEmpty(t, payload.Data)
	})

	t.Run("success - hierarchical event type", func(t *testing.T) {
		payload, err := New("order.item.updated", map[string]string{"id": "123"})
		require.NoError(t, err)
		assert.Equal(t, "order.item.updated", payload.Type)
	})

	t.Run("error - invalid event type format", func(t *testing.T) {
		_, err := New("invalid-type-with-dashes", map[string]string{"id": "123"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validating payload")
	})

	t.Run("error - empty event type", func(t *testing.T) {
		_, err := New("", map[string]string{"id": "123"})
		require.Error(t, err)
	})

	t.Run("error - data cannot be marshaled", func(t *testing.T) {
		// channels cannot be marshaled to JSON
		_, err := New("test.event", make(chan int))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "marshaling data")
	})
}

func TestParse(t *testing.T) {
	t.Run("success - valid payload", func(t *testing.T) {
		data := []byte(`{
			"type": "user.created",
			"timestamp": "2024-01-01T12:00:00Z",
			"data": {"user_id": 123, "name": "John Doe"}
		}`)

		payload, err := Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "user.created", payload.Type)
		assert.Equal(t, 2024, payload.Timestamp.Year())
		assert.NotEmpty(t, payload.Data)
	})

	t.Run("success - timestamp with nanoseconds", func(t *testing.T) {
		data := []byte(`{
			"type": "test.event",
			"timestamp": "2024-01-01T12:00:00.123456789Z",
			"data": {"foo": "bar"}
		}`)

		payload, err := Parse(data)
		require.NoError(t, err)
		assert.NotZero(t, payload.Timestamp.Nanosecond())
	})

	t.Run("error - invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		_, err := Parse(data)
		require.Error(t, err)
	})

	t.Run("error - missing type", func(t *testing.T) {
		data := []byte(`{
			"timestamp": "2024-01-01T12:00:00Z",
			"data": {"foo": "bar"}
		}`)

		_, err := Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("error - missing timestamp", func(t *testing.T) {
		data := []byte(`{
			"type": "test.event",
			"data": {"foo": "bar"}
		}`)

		_, err := Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp")
	})

	t.Run("error - missing data", func(t *testing.T) {
		data := []byte(`{
			"type": "test.event",
			"timestamp": "2024-01-01T12:00:00Z"
		}`)

		_, err := Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data is required")
	})

	t.Run("error - invalid event type format", func(t *testing.T) {
		data := []byte(`{
			"type": "invalid-type",
			"timestamp": "2024-01-01T12:00:00Z",
			"data": {"foo": "bar"}
		}`)

		_, err := Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hierarchical")
	})
}

func TestValidate(t *testing.T) {
	t.Run("success - valid payload", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "user.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"user_id": 123}`),
		}

		err := payload.Validate()
		require.NoError(t, err)
	})

	t.Run("error - empty type", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"user_id": 123}`),
		}

		err := payload.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("error - invalid type format", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "invalid@type",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"user_id": 123}`),
		}

		err := payload.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hierarchical")
	})

	t.Run("error - zero timestamp", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "user.created",
			Timestamp: time.Time{},
			Data:      json.RawMessage(`{"user_id": 123}`),
		}

		err := payload.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp is required")
	})

	t.Run("error - empty data", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "user.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(``),
		}

		err := payload.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data is required")
	})

	t.Run("error - invalid JSON data", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "user.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{invalid}`),
		}

		err := payload.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data must be valid JSON")
	})
}

func TestBytes(t *testing.T) {
	t.Run("success - returns JSON bytes", func(t *testing.T) {
		payload, err := New("test.event", map[string]string{"foo": "bar"})
		require.NoError(t, err)

		bytes, err := payload.Bytes()
		require.NoError(t, err)
		assert.NotEmpty(t, bytes)

		// Should be valid JSON
		var decoded StandardPayload
		err = json.Unmarshal(bytes, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "test.event", decoded.Type)
	})
}

func TestMatchesEventType(t *testing.T) {
	payload, err := New("user.created", map[string]string{"id": "123"})
	require.NoError(t, err)

	t.Run("success - exact match", func(t *testing.T) {
		matches := payload.MatchesEventType([]string{"user.created"})
		assert.True(t, matches)
	})

	t.Run("success - prefix match with wildcard", func(t *testing.T) {
		matches := payload.MatchesEventType([]string{"user.*"})
		assert.True(t, matches)
	})

	t.Run("success - multiple filters, one matches", func(t *testing.T) {
		matches := payload.MatchesEventType([]string{"order.*", "user.*", "product.*"})
		assert.True(t, matches)
	})

	t.Run("success - empty filter accepts all", func(t *testing.T) {
		matches := payload.MatchesEventType([]string{})
		assert.True(t, matches)
	})

	t.Run("failure - no match", func(t *testing.T) {
		matches := payload.MatchesEventType([]string{"order.created"})
		assert.False(t, matches)
	})

	t.Run("failure - prefix doesn't match", func(t *testing.T) {
		matches := payload.MatchesEventType([]string{"order.*"})
		assert.False(t, matches)
	})

	t.Run("failure - partial prefix doesn't match", func(t *testing.T) {
		// "us.*" should NOT match "user.created"
		matches := payload.MatchesEventType([]string{"us.*"})
		assert.False(t, matches)
	})
}

func TestValidateEventType(t *testing.T) {
	t.Run("success - simple type", func(t *testing.T) {
		err := ValidateEventType("user")
		require.NoError(t, err)
	})

	t.Run("success - hierarchical type", func(t *testing.T) {
		err := ValidateEventType("user.created")
		require.NoError(t, err)
	})

	t.Run("success - deeply nested type", func(t *testing.T) {
		err := ValidateEventType("order.item.inventory.updated")
		require.NoError(t, err)
	})

	t.Run("success - with underscores and numbers", func(t *testing.T) {
		err := ValidateEventType("user_v2.profile_123.updated")
		require.NoError(t, err)
	})

	t.Run("success - wildcard suffix", func(t *testing.T) {
		err := ValidateEventType("user.*")
		require.NoError(t, err)
	})

	t.Run("error - empty type", func(t *testing.T) {
		err := ValidateEventType("")
		require.Error(t, err)
	})

	t.Run("error - contains dashes", func(t *testing.T) {
		err := ValidateEventType("user-created")
		require.Error(t, err)
	})

	t.Run("error - contains special characters", func(t *testing.T) {
		err := ValidateEventType("user@created")
		require.Error(t, err)
	})

	t.Run("error - starts with period", func(t *testing.T) {
		err := ValidateEventType(".user.created")
		require.Error(t, err)
	})

	t.Run("error - ends with period (without wildcard)", func(t *testing.T) {
		err := ValidateEventType("user.")
		require.Error(t, err)
	})

	t.Run("error - double periods", func(t *testing.T) {
		err := ValidateEventType("user..created")
		require.Error(t, err)
	})
}

func TestMarshalUnmarshal(t *testing.T) {
	t.Run("round-trip - preserves data", func(t *testing.T) {
		original := StandardPayload{
			Type:      "test.event",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 123456789, time.UTC),
			Data:      json.RawMessage(`{"foo":"bar","num":123}`),
		}

		// Marshal
		bytes, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal
		var decoded StandardPayload
		err = json.Unmarshal(bytes, &decoded)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original.Type, decoded.Type)
		assert.True(t, original.Timestamp.Equal(decoded.Timestamp))
		assert.JSONEq(t, string(original.Data), string(decoded.Data))
	})

	t.Run("timestamp format - ISO 8601", func(t *testing.T) {
		payload := StandardPayload{
			Type:      "test.event",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Data:      json.RawMessage(`{"foo":"bar"}`),
		}

		bytes, err := json.Marshal(payload)
		require.NoError(t, err)

		// Check timestamp is in ISO 8601 format
		var raw map[string]interface{}
		err = json.Unmarshal(bytes, &raw)
		require.NoError(t, err)
		assert.Contains(t, raw["timestamp"], "2024-01-01T12:00:00")
	})
}
