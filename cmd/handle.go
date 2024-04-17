package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// tokenInfo struct represents the structure of JWT token information
type tokenInfo struct {
	Header    interface{} `json:"header"`    // Header of the JWT token
	Payload   interface{} `json:"payload"`   // Payload (claims) of the JWT token
	Signature string      `json:"signature"` // Signature of the JWT token
}

// ParseFile reads tokens from a file and returns them as a slice
func ParseFile(fn string) ([]string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("[-] File provided not found\n")
	}
	defer f.Close()

	var tokens []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			tokens = append(tokens, line)
		}
	}
	return tokens, nil
}

// ParseTokenInfo parses the header, payload, and signature of a JWT token
// and returns the token info as a JSON string.
func ParseTokenInfo(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	headerBytes, err := jwt.DecodeSegment(parts[0])
	if err != nil {
		return "", fmt.Errorf("error decoding header: %v", err)
	}
	var header interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", fmt.Errorf("error unmarshalling header: %v", err)
	}

	payloadBytes, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		return "", fmt.Errorf("error decoding payload: %v", err)
	}
	var payload interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", fmt.Errorf("error unmarshalling payload: %v", err)
	}

	signature := parts[2]

	tokenInfo := tokenInfo{
		Header:    header,
		Payload:   payload,
		Signature: signature,
	}

	tokenInfoJSON, err := json.MarshalIndent(tokenInfo, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error encoding token info to JSON: %v", err)
	}

	return string(tokenInfoJSON), nil
}
