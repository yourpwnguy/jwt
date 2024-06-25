// package runner

// import (
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"strings"

// 	"github.com/dgrijalva/jwt-go"
// )

// // tokenInfo struct represents the structure of JWT token information
// type tokenInfo struct {
// 	Header    interface{} `json:"header"`    // Header of the JWT token
// 	Payload   interface{} `json:"payload"`   // Payload (claims) of the JWT token
// 	Signature string      `json:"signature"` // Signature of the JWT token
// }

// // ParseTokenInfo parses the header, payload, and signature of a JWT token
// // and returns the token info as a JSON string.
// func ExtractTokenInfo(token string) (string, error) {
// 	parts := strings.Split(token, ".")
// 	if len(parts) != 3 {
// 		return "", fmt.Errorf("invalid token format")
// 	}

// 	for _, part := range parts {
// 		headerBytes, err := jwt.DecodeSegment(part)
// 		if err != nil {
// 			return "", fmt.Errorf("error decoding header: %v", err)
// 		}
// 		var header interface{}
// 		if err := json.Unmarshal(headerBytes, &header); err != nil {
// 			return "", fmt.Errorf("error unmarshalling header: %v", err)
// 		}

// 		payloadBytes, err := jwt.DecodeSegment(parts[1])
// 		if err != nil {
// 			return "", fmt.Errorf("error decoding payload: %v", err)
// 		}
// 		var payload interface{}
// 		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
// 			return "", fmt.Errorf("error unmarshalling payload: %v", err)
// 		}

// 		signature := parts[2]

// 		tokenInfo := tokenInfo{
// 			Header:    header,
// 			Payload:   payload,
// 			Signature: signature,
// 		}

// 		tokenInfoJSON, err := json.MarshalIndent(tokenInfo, "", "  ")
// 		if err != nil {
// 			return "", fmt.Errorf("error encoding token info to JSON: %v", err)
// 		}
// 	}

// 	return string(tokenInfoJSON), nil
// }

// // Function to decode the base64 encoded jwt token
// func DecodeB64Tokn(tokn string) (string, error) {
// 	base64, err := base64.StdEncoding.DecodeString(tokn)
// 	if err != nil {
// 		return "", fmt.Errorf("unable to decode the string")
// 	}
// 	return string(base64), nil
// }

package runner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// tokenInfo struct represents the structure of JWT token information
type tokenInfo struct {
	Header    interface{} `json:"header"`    // Header of the JWT token
	Payload   interface{} `json:"payload"`   // Payload (claims) of the JWT token
	Signature string      `json:"signature"` // Signature of the JWT token
}

// ExtractTokenInfo parses the header, payload, and signature of a JWT token
// and returns the token info as a JSON string.
func ExtractTokenInfo(token string) (string, error) {

	// We split the token into 3 parts ( header, payload, signature )
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// Variables for holding the parts values
	var (
		header, payload interface{}
		signature       string
	)
	
	// Attempt to decode and unmarshal the header
	headerBytes, err := jwt.DecodeSegment(parts[0])
	if err != nil {
		fmt.Printf(Errfix+" error decoding header: %v\n", err)
	} else if err := json.Unmarshal(headerBytes, &header); err != nil {
		fmt.Printf(Errfix+" error unmarshalling header: %v\n", err)
	}

	// Attempt to decode and unmarshal the payload
	payloadBytes, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		fmt.Printf(Errfix+" error decoding payload: %v\n", err)
	} else if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		fmt.Printf(Errfix+" error unmarshalling payload: %v\n", err)
	}

	// Set the signature
	signature = parts[2]

	// Instantiating the options struct and setting values
	tokenInfo := tokenInfo{
		Header:    header,
		Payload:   payload,
		Signature: signature,
	}

	// Translating the struct to JSON Bytes
	tokenInfoJSON, err := json.MarshalIndent(tokenInfo, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error encoding token info to JSON: %v", err)
	}

	// Returning the string version of JSON
	return string(tokenInfoJSON), nil
}

// DecodeB64Tokn decodes the base64 encoded JWT token
func DecodeB64Tokn(tokn string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(tokn)
	if err != nil {
		return "", fmt.Errorf("unable to decode the string")
	}
	return string(decoded), nil
}
