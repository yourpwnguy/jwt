package jwt

import (
	"encoding/base64"
	"testing"
)

// encode_test.go pins the one property that matters: what you give
// Encode is exactly what ends up in the token. No JSON parsing, no
// key sorting, no whitespace cleanup.

func TestEncodeExactBytes(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		payload string
		sig     string
	}{
		{
			name:    "minimal",
			header:  `{"alg":"HS256"}`,
			payload: `{"sub":"1"}`,
			sig:     "abc",
		},
		{
			name:    "whitespace is preserved",
			header:  "{\n  \"alg\": \"HS256\"\n}",
			payload: "{\n  \"sub\": \"1\"\n}",
			sig:     "",
		},
		{
			name:    "key order is preserved",
			header:  `{"typ":"JWT","alg":"HS256"}`,
			payload: `{"b":2,"a":1}`,
			sig:     "sig",
		},
		{
			name:    "duplicate keys survive",
			header:  `{"alg":"HS256"}`,
			payload: `{"sub":"first","sub":"second"}`,
			sig:     "sig",
		},
		{
			name:    "unicode survives",
			header:  `{"alg":"HS256"}`,
			payload: `{"name":"Jöhn"}`,
			sig:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Encode([]byte(tt.header), []byte(tt.payload), tt.sig)

			wantHeader := base64.RawURLEncoding.EncodeToString([]byte(tt.header))
			wantPayload := base64.RawURLEncoding.EncodeToString([]byte(tt.payload))
			want := wantHeader + "." + wantPayload + "." + tt.sig
			if got != want {
				t.Fatalf("Encode() = %q, want %q", got, want)
			}

			// Decode the segments and compare the bytes directly. This
			// catches accidental normalization even if the token shape
			// still looks correct.
			parts := splitToken(t, got)
			if string(decodeSegment(t, parts[0])) != tt.header {
				t.Errorf("header round trip changed bytes")
			}
			if string(decodeSegment(t, parts[1])) != tt.payload {
				t.Errorf("payload round trip changed bytes")
			}
		})
	}
}

func TestEncodeEmptySignature(t *testing.T) {
	got := Encode([]byte(`{"alg":"none"}`), []byte(`{}`), "")
	if got[len(got)-1] != '.' {
		t.Fatalf("unsigned token must end in a dot, got %q", got)
	}
}

func TestSigningInputMatchesEncode(t *testing.T) {
	header := []byte(`{"alg":"HS256"}`)
	payload := []byte(`{"sub":"1"}`)
	input := SigningInput(header, payload)
	token := Encode(header, payload, "sig")

	want := input + ".sig"
	if token != want {
		t.Fatalf("Encode() = %q, want %q", token, want)
	}
}

// splitToken and decodeSegment keep the assertions obvious without
// pulling helpers from parser_test.go.
func splitToken(t *testing.T, token string) []string {
	t.Helper()
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3: %q", len(parts), token)
	}
	return parts
}

func decodeSegment(t *testing.T, seg string) []byte {
	t.Helper()
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		t.Fatalf("decode segment: %v", err)
	}
	return data
}
