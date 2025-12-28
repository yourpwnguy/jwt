package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for JWT parsing operations.
//
// These errors indicate specific failure modes during token parsing.
// They can be checked using errors.Is() and may be wrapped with
// additional context.
var (
	ErrInvalidFormat    = errors.New("invalid token format")
	ErrDecodeHeader     = errors.New("failed to decode header")
	ErrDecodePayload    = errors.New("failed to decode payload")
	ErrUnmarshalHeader  = errors.New("failed to unmarshal header")
	ErrUnmarshalPayload = errors.New("failed to unmarshal payload")
)

// Token segment indices for clarity and maintainability.
const (
	expectedParts = 3
	partHeader    = 0
	partPayload   = 1
	partSignature = 2
)

// Parse decodes a JWT token string and returns a formatted JSON representation.
//
// The function performs the following steps:
//  1. Splits the token into its three segments
//  2. Decodes and unmarshals the header (base64url → JSON)
//  3. Decodes and unmarshals the payload (base64url → JSON)
//  4. Preserves the signature as-is (it's binary, not JSON)
//  5. Returns the combined structure as pretty-printed JSON
//
// Parameters:
//   - raw: The complete JWT token string (header.payload.signature)
//
// Returns:
//   - A pretty-printed JSON string representing the decoded token
//   - An error if the token is malformed or cannot be decoded
//
// Note: This function does NOT verify the token's signature.
// It is intended for inspection and debugging purposes only.
func Parse(raw string) (string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != expectedParts {
		return "", fmt.Errorf("%w: expected %d parts, got %d",
			ErrInvalidFormat, expectedParts, len(parts))
	}

	header, err := decodeSegment(parts[partHeader])
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecodeHeader, err)
	}

	payload, err := decodeSegment(parts[partPayload])
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecodePayload, err)
	}

	token := Token{
		Header:    header,
		Payload:   payload,
		Signature: parts[partSignature],
	}

	return toJSON(token)
}

// decodeSegment decodes a single base64url-encoded JWT segment.
//
// JWT segments use base64url encoding (RFC 4648) without padding.
// This function handles the decoding and JSON unmarshaling in one step.
//
// Parameters:
//   - seg: A base64url-encoded JSON string (without padding)
//
// Returns:
//   - The decoded JSON as an any type (preserves structure)
//   - An error if decoding or unmarshaling fails
func decodeSegment(seg string) (any, error) {
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return nil, err
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// toJSON converts a Token struct to a pretty-printed JSON string.
//
// The output uses 2-space indentation for readability.
//
// Parameters:
//   - token: The Token struct to serialize
//
// Returns:
//   - A formatted JSON string
//   - An error if JSON marshaling fails (should not happen with valid Token)
func toJSON(token Token) (string, error) {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}
	return string(data), nil
}
