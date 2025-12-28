package encoding

import (
	"encoding/base64"
	"errors"
)

// ErrInvalidBase64 indicates that the input string is not valid
// base64-encoded data.
//
// This error is returned when DecodeBase64 fails to decode the input.
var ErrInvalidBase64 = errors.New("invalid base64 encoding")

// DecodeBase64 decodes a standard base64-encoded string.
//
// This function uses Go's standard base64 encoding (RFC 4648),
// which includes padding characters (=). It is useful for handling
// JWT tokens that may have been additionally base64-encoded for
// transport (double-encoding).
//
// Parameters:
//   - s: A base64-encoded string (with padding)
//
// Returns:
//   - The decoded string
//   - ErrInvalidBase64 if the input is not valid base64
func DecodeBase64(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", ErrInvalidBase64
	}
	return string(decoded), nil
}
