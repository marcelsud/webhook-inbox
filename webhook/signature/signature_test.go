package signature

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecret(t *testing.T) {
	t.Run("success - minimum size", func(t *testing.T) {
		secret, err := GenerateSecret(MinSecretBytes)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(secret.String(), SecretPrefix))
		assert.Equal(t, MinSecretBytes, len(secret.Bytes()))
	})

	t.Run("success - maximum size", func(t *testing.T) {
		secret, err := GenerateSecret(MaxSecretBytes)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(secret.String(), SecretPrefix))
		assert.Equal(t, MaxSecretBytes, len(secret.Bytes()))
	})

	t.Run("success - medium size", func(t *testing.T) {
		secret, err := GenerateSecret(32)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(secret.String(), SecretPrefix))
		assert.Equal(t, 32, len(secret.Bytes()))
	})

	t.Run("error - too small", func(t *testing.T) {
		_, err := GenerateSecret(MinSecretBytes - 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret size must be between")
	})

	t.Run("error - too large", func(t *testing.T) {
		_, err := GenerateSecret(MaxSecretBytes + 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret size must be between")
	})

	t.Run("randomness - generates different secrets", func(t *testing.T) {
		secret1, err1 := GenerateSecret(32)
		secret2, err2 := GenerateSecret(32)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, secret1.String(), secret2.String())
	})
}

func TestParseSecret(t *testing.T) {
	t.Run("success - valid secret", func(t *testing.T) {
		// Generate a secret first
		original, err := GenerateSecret(32)
		require.NoError(t, err)

		// Parse it back
		parsed, err := ParseSecret(original.String())
		require.NoError(t, err)
		assert.Equal(t, original.String(), parsed.String())
		assert.Equal(t, original.Bytes(), parsed.Bytes())
	})

	t.Run("error - missing prefix", func(t *testing.T) {
		_, err := ParseSecret("dGVzdHNlY3JldA==")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must start with")
	})

	t.Run("error - invalid base64", func(t *testing.T) {
		_, err := ParseSecret(SecretPrefix + "not-valid-base64!!!")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding base64")
	})

	t.Run("error - secret too small", func(t *testing.T) {
		// Generate a base64 string that's too small
		smallSecret := SecretPrefix + "dGVzdA==" // "test" in base64 (4 bytes)
		_, err := ParseSecret(smallSecret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret size must be between")
	})
}

func TestSign(t *testing.T) {
	secret, err := GenerateSecret(32)
	require.NoError(t, err)

	msgID := "msg_test123"
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	payload := []byte(`{"type":"test.event","timestamp":"2024-01-01T12:00:00Z","data":{"foo":"bar"}}`)

	t.Run("success - creates valid signature", func(t *testing.T) {
		sig, err := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err)
		assert.Equal(t, SignatureVersion, sig.Version)
		assert.NotEmpty(t, sig.Signature)
		assert.True(t, strings.HasPrefix(sig.String(), "v1,"))
	})

	t.Run("success - same inputs produce same signature", func(t *testing.T) {
		sig1, err1 := Sign(secret, msgID, timestamp, payload)
		sig2, err2 := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, sig1.String(), sig2.String())
	})

	t.Run("success - different inputs produce different signatures", func(t *testing.T) {
		sig1, err1 := Sign(secret, msgID, timestamp, payload)
		sig2, err2 := Sign(secret, "msg_different", timestamp, payload)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, sig1.String(), sig2.String())
	})

	t.Run("error - message ID contains period", func(t *testing.T) {
		_, err := Sign(secret, "msg.with.periods", timestamp, payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not contain '.'")
	})
}

func TestVerify(t *testing.T) {
	secret, err := GenerateSecret(32)
	require.NoError(t, err)

	msgID := "msg_test123"
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	payload := []byte(`{"type":"test.event","timestamp":"2024-01-01T12:00:00Z","data":{"foo":"bar"}}`)

	t.Run("success - valid signature", func(t *testing.T) {
		sig, err := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err)

		valid, err := Verify(secret, msgID, timestamp, payload, sig)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("failure - wrong secret", func(t *testing.T) {
		sig, err := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err)

		wrongSecret, err := GenerateSecret(32)
		require.NoError(t, err)

		valid, err := Verify(wrongSecret, msgID, timestamp, payload, sig)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("failure - wrong message ID", func(t *testing.T) {
		sig, err := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err)

		valid, err := Verify(secret, "msg_wrong", timestamp, payload, sig)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("failure - wrong timestamp", func(t *testing.T) {
		sig, err := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err)

		wrongTime := timestamp.Add(1 * time.Hour)
		valid, err := Verify(secret, msgID, wrongTime, payload, sig)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("failure - wrong payload", func(t *testing.T) {
		sig, err := Sign(secret, msgID, timestamp, payload)
		require.NoError(t, err)

		wrongPayload := []byte(`{"type":"different.event","timestamp":"2024-01-01T12:00:00Z","data":{"foo":"baz"}}`)
		valid, err := Verify(secret, msgID, timestamp, wrongPayload, sig)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("error - unsupported version", func(t *testing.T) {
		sig := Signature{
			Version:   "v2",
			Signature: "dGVzdA==",
		}

		_, err := Verify(secret, msgID, timestamp, payload, sig)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported signature version")
	})
}

func TestVerifyMultiple(t *testing.T) {
	secret1, err := GenerateSecret(32)
	require.NoError(t, err)
	secret2, err := GenerateSecret(32)
	require.NoError(t, err)

	msgID := "msg_test123"
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	payload := []byte(`{"type":"test.event","timestamp":"2024-01-01T12:00:00Z","data":{"foo":"bar"}}`)

	t.Run("success - verifies with first secret", func(t *testing.T) {
		sig, err := Sign(secret1, msgID, timestamp, payload)
		require.NoError(t, err)

		valid, err := VerifyMultiple([]Secret{secret1, secret2}, msgID, timestamp, payload, []Signature{sig})
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("success - verifies with second secret", func(t *testing.T) {
		sig, err := Sign(secret2, msgID, timestamp, payload)
		require.NoError(t, err)

		valid, err := VerifyMultiple([]Secret{secret1, secret2}, msgID, timestamp, payload, []Signature{sig})
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("success - multiple signatures, one matches", func(t *testing.T) {
		sig1, err := Sign(secret1, msgID, timestamp, payload)
		require.NoError(t, err)
		sig2, err := Sign(secret2, msgID, timestamp, payload)
		require.NoError(t, err)

		valid, err := VerifyMultiple([]Secret{secret1}, msgID, timestamp, payload, []Signature{sig1, sig2})
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("failure - no matching secret", func(t *testing.T) {
		sig, err := Sign(secret1, msgID, timestamp, payload)
		require.NoError(t, err)

		secret3, err := GenerateSecret(32)
		require.NoError(t, err)

		valid, err := VerifyMultiple([]Secret{secret2, secret3}, msgID, timestamp, payload, []Signature{sig})
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("error - no secrets", func(t *testing.T) {
		sig, err := Sign(secret1, msgID, timestamp, payload)
		require.NoError(t, err)

		_, err = VerifyMultiple([]Secret{}, msgID, timestamp, payload, []Signature{sig})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide at least one")
	})

	t.Run("error - no signatures", func(t *testing.T) {
		_, err := VerifyMultiple([]Secret{secret1}, msgID, timestamp, payload, []Signature{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide at least one")
	})
}

func TestParseSignature(t *testing.T) {
	t.Run("success - valid signature", func(t *testing.T) {
		sig, err := ParseSignature("v1,dGVzdHNpZ25hdHVyZQ==")
		require.NoError(t, err)
		assert.Equal(t, "v1", sig.Version)
		assert.Equal(t, "dGVzdHNpZ25hdHVyZQ==", sig.Signature)
	})

	t.Run("error - invalid format", func(t *testing.T) {
		_, err := ParseSignature("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature format")
	})

	t.Run("error - empty string", func(t *testing.T) {
		_, err := ParseSignature("")
		require.Error(t, err)
	})
}

func TestParseSignatureHeader(t *testing.T) {
	t.Run("success - single signature", func(t *testing.T) {
		sigs, err := ParseSignatureHeader("v1,dGVzdA==")
		require.NoError(t, err)
		assert.Len(t, sigs, 1)
		assert.Equal(t, "v1", sigs[0].Version)
	})

	t.Run("success - multiple signatures", func(t *testing.T) {
		sigs, err := ParseSignatureHeader("v1,dGVzdA== v1a,YW5vdGhlcg==")
		require.NoError(t, err)
		assert.Len(t, sigs, 2)
		assert.Equal(t, "v1", sigs[0].Version)
		assert.Equal(t, "v1a", sigs[1].Version)
	})

	t.Run("success - extra whitespace", func(t *testing.T) {
		sigs, err := ParseSignatureHeader("  v1,dGVzdA==   v1a,YW5vdGhlcg==  ")
		require.NoError(t, err)
		assert.Len(t, sigs, 2)
	})

	t.Run("error - empty header", func(t *testing.T) {
		_, err := ParseSignatureHeader("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("error - invalid signature format", func(t *testing.T) {
		_, err := ParseSignatureHeader("invalid")
		require.Error(t, err)
	})
}

func TestBuildSignatureHeader(t *testing.T) {
	t.Run("success - single signature", func(t *testing.T) {
		sig := Signature{Version: "v1", Signature: "dGVzdA=="}
		header := BuildSignatureHeader([]Signature{sig})
		assert.Equal(t, "v1,dGVzdA==", header)
	})

	t.Run("success - multiple signatures", func(t *testing.T) {
		sig1 := Signature{Version: "v1", Signature: "dGVzdA=="}
		sig2 := Signature{Version: "v1a", Signature: "YW5vdGhlcg=="}
		header := BuildSignatureHeader([]Signature{sig1, sig2})
		assert.Equal(t, "v1,dGVzdA== v1a,YW5vdGhlcg==", header)
	})
}
