package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Segment positions, named so "parts[1]" never appears in logic.
const (
	expectedParts = 3
	partHeader    = 0
	partPayload   = 1
	partSignature = 2
)

// Parse decodes a clean JWT string into a Token.
//
// It does NOT verify the signature — decoding tells you what's inside,
// verification tells you whether to trust it. Mixing those two is how
// people get owned, so they live in different packages on purpose.
//
// The input must already be trimmed (no "Bearer ", no quotes). See
// internal/input for that cleaning step.
func Parse(raw string) (Token, error) {
	if len(raw) > MaxTokenSize {
		return Token{}, fmt.Errorf("%w: %d bytes > %d", ErrTooLarge, len(raw), MaxTokenSize)
	}
	// TrimSpace here is just belt-and-braces; input should have done it.
	// Keeping it makes Parse forgiving when called directly in tests.
	raw = strings.TrimSpace(raw)

	parts := strings.Split(raw, ".")
	if len(parts) != expectedParts {
		// Hint JWE explicitly — otherwise users stare at "expected 3 got 5".
		if len(parts) == 5 {
			return Token{}, fmt.Errorf("%w: got 5 parts (this looks like JWE, only JWT is supported)", ErrInvalidFormat)
		}
		return Token{}, fmt.Errorf("%w: expected %d parts, got %d", ErrInvalidFormat, expectedParts, len(parts))
	}

	header, err := decodeObject(parts[partHeader])
	if err != nil {
		return Token{}, fmt.Errorf("%w: %v", ErrDecodeHeader, err)
	}
	payload, err := decodeObject(parts[partPayload])
	if err != nil {
		return Token{}, fmt.Errorf("%w: %v", ErrDecodePayload, err)
	}

	return Token{
		Header:       header,
		Payload:      payload,
		Signature:    parts[partSignature],
		SigningInput: parts[partHeader] + "." + parts[partPayload],
		Raw:          raw,
	}, nil
}

// decodeObject decodes one base64url JSON-object segment.
//
// Why RawURLEncoding and not StdEncoding? RFC 7519 mandates base64url
// without padding. Accepting StdEncoding here would silently accept
// non-JWT strings and produce confusing errors downstream.
func decodeObject(seg string) (map[string]any, error) {
	if len(seg) == 0 {
		return nil, fmt.Errorf("empty segment")
	}
	if len(seg) > MaxSegmentSize {
		return nil, fmt.Errorf("%w: segment %d bytes", ErrTooLarge, len(seg))
	}
	// Performance note: DecodeString allocates once for the JSON bytes.
	// That's fine — segments are small and this keeps the code obvious.
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("segment is JSON null, want object")
	}
	return obj, nil
}
