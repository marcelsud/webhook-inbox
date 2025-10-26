package signature

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// SecretPrefix is the prefix for Standard Webhooks symmetric secrets
	SecretPrefix = "whsec_"

	// SignatureVersion is the version identifier for symmetric signatures
	SignatureVersion = "v1"

	// MinSecretBytes is the minimum recommended secret size (192 bits)
	MinSecretBytes = 24

	// MaxSecretBytes is the maximum recommended secret size (512 bits)
	MaxSecretBytes = 64
)

// Secret represents a Standard Webhooks signing secret
type Secret struct {
	raw    []byte
	base64 string
}

// GenerateSecret creates a new cryptographically secure signing secret
// between MinSecretBytes and MaxSecretBytes in size.
func GenerateSecret(size int) (Secret, error) {
	if size < MinSecretBytes || size > MaxSecretBytes {
		return Secret{}, fmt.Errorf("secret size must be between %d and %d bytes", MinSecretBytes, MaxSecretBytes)
	}

	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return Secret{}, fmt.Errorf("generating random bytes: %w", err)
	}

	return Secret{
		raw:    bytes,
		base64: SecretPrefix + base64.StdEncoding.EncodeToString(bytes),
	}, nil
}

// ParseSecret parses a base64-encoded secret with the whsec_ prefix
func ParseSecret(encoded string) (Secret, error) {
	if !strings.HasPrefix(encoded, SecretPrefix) {
		return Secret{}, fmt.Errorf("secret must start with %s prefix", SecretPrefix)
	}

	b64 := strings.TrimPrefix(encoded, SecretPrefix)
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return Secret{}, fmt.Errorf("decoding base64 secret: %w", err)
	}

	if len(raw) < MinSecretBytes || len(raw) > MaxSecretBytes {
		return Secret{}, fmt.Errorf("secret size must be between %d and %d bytes", MinSecretBytes, MaxSecretBytes)
	}

	return Secret{
		raw:    raw,
		base64: encoded,
	}, nil
}

// String returns the base64-encoded secret with prefix
func (s Secret) String() string {
	return s.base64
}

// Bytes returns the raw secret bytes
func (s Secret) Bytes() []byte {
	return s.raw
}

// Signature represents a signed webhook with its signature
type Signature struct {
	Version   string
	Signature string
}

// String returns the signature in the format: v1,<base64_signature>
func (s Signature) String() string {
	return fmt.Sprintf("%s,%s", s.Version, s.Signature)
}

// ParseSignature parses a signature string in the format: v1,<base64_signature>
func ParseSignature(sig string) (Signature, error) {
	parts := strings.SplitN(sig, ",", 2)
	if len(parts) != 2 {
		return Signature{}, fmt.Errorf("invalid signature format, expected 'version,signature'")
	}

	return Signature{
		Version:   parts[0],
		Signature: parts[1],
	}, nil
}

// Sign creates a Standard Webhooks signature for the given webhook
// The signed content is: {msgID}.{timestamp}.{payload}
func Sign(secret Secret, msgID string, timestamp time.Time, payload []byte) (Signature, error) {
	// Validate inputs
	if strings.Contains(msgID, ".") {
		return Signature{}, fmt.Errorf("message ID must not contain '.'")
	}

	// Create the signed content: msgID.timestamp.payload
	timestampStr := strconv.FormatInt(timestamp.Unix(), 10)
	signedContent := fmt.Sprintf("%s.%s.%s", msgID, timestampStr, payload)

	// Create HMAC-SHA256 signature
	mac := hmac.New(sha256.New, secret.Bytes())
	mac.Write([]byte(signedContent))
	signature := mac.Sum(nil)

	return Signature{
		Version:   SignatureVersion,
		Signature: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// Verify verifies a webhook signature using constant-time comparison
// Returns true if the signature is valid, false otherwise
func Verify(secret Secret, msgID string, timestamp time.Time, payload []byte, expectedSig Signature) (bool, error) {
	// Only support v1 signatures
	if expectedSig.Version != SignatureVersion {
		return false, fmt.Errorf("unsupported signature version: %s", expectedSig.Version)
	}

	// Generate the expected signature
	calculatedSig, err := Sign(secret, msgID, timestamp, payload)
	if err != nil {
		return false, fmt.Errorf("calculating signature: %w", err)
	}

	// Decode both signatures for constant-time comparison
	expected, err := base64.StdEncoding.DecodeString(expectedSig.Signature)
	if err != nil {
		return false, fmt.Errorf("decoding expected signature: %w", err)
	}

	calculated, err := base64.StdEncoding.DecodeString(calculatedSig.Signature)
	if err != nil {
		return false, fmt.Errorf("decoding calculated signature: %w", err)
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(expected, calculated) == 1, nil
}

// VerifyMultiple verifies a webhook against multiple signatures (for secret rotation)
// Returns true if any of the signatures is valid
func VerifyMultiple(secrets []Secret, msgID string, timestamp time.Time, payload []byte, signatures []Signature) (bool, error) {
	if len(secrets) == 0 || len(signatures) == 0 {
		return false, fmt.Errorf("must provide at least one secret and one signature")
	}

	// Try each signature against each secret
	for _, sig := range signatures {
		for _, secret := range secrets {
			valid, err := Verify(secret, msgID, timestamp, payload, sig)
			if err != nil {
				// Log error but continue trying other combinations
				continue
			}
			if valid {
				return true, nil
			}
		}
	}

	return false, nil
}

// ParseSignatureHeader parses the webhook-signature header which contains
// space-delimited signatures: "v1,sig1 v1,sig2"
func ParseSignatureHeader(header string) ([]Signature, error) {
	if header == "" {
		return nil, fmt.Errorf("signature header is empty")
	}

	parts := strings.Split(header, " ")
	signatures := make([]Signature, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		sig, err := ParseSignature(part)
		if err != nil {
			return nil, fmt.Errorf("parsing signature '%s': %w", part, err)
		}

		signatures = append(signatures, sig)
	}

	if len(signatures) == 0 {
		return nil, fmt.Errorf("no valid signatures found in header")
	}

	return signatures, nil
}

// BuildSignatureHeader builds the webhook-signature header value
// from multiple signatures (space-delimited)
func BuildSignatureHeader(signatures []Signature) string {
	parts := make([]string, len(signatures))
	for i, sig := range signatures {
		parts[i] = sig.String()
	}
	return strings.Join(parts, " ")
}
