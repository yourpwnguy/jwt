package jwt

import "encoding/base64"

// encode.go is the write side of the token format. Decode lives in
// parser.go; this file only assembles bytes.
//
// Why not marshal a map back to JSON? Because that would silently
// rewrite the token. Map marshaling reorders keys, drops duplicate
// keys, and changes whitespace. Anyone testing parser differences
// between services needs the exact bytes they typed, so Encode takes
// raw JSON and never interprets it.
//
// Validation happens before this point, in the TUI or the caller. A
// malformed segment produces a malformed token on purpose; that can be
// useful when you are probing a server.

// Encode assembles "header.payload.signature" from exact JSON bytes.
//
// Header and payload are base64url encoded without padding, as RFC
// 7515 requires. Signature is expected to already be base64url text;
// pass "" for an unsigned token. Encode does not verify or validate
// either segment.
func Encode(headerJSON, payloadJSON []byte, signature string) string {
	return SigningInput(headerJSON, payloadJSON) + "." + signature
}

// SigningInput returns the exact "header.payload" string that a
// signature covers. Keep this separate from Encode so signing code
// cannot accidentally sign a different encoding than the token it
// prints.
func SigningInput(headerJSON, payloadJSON []byte) string {
	return base64.RawURLEncoding.EncodeToString(headerJSON) +
		"." +
		base64.RawURLEncoding.EncodeToString(payloadJSON)
}
