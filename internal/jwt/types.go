package jwt

// Token represents the decoded structure of a JSON Web Token.
//
// Each field corresponds to one of the three segments of a JWT.
// The Header and Payload fields are stored as any types to preserve
// the original JSON structure, which may contain nested objects.
//
// JSON output example:
//
//	{
//	  "header": {
//	    "alg": "HS256",
//	    "typ": "JWT"
//	  },
//	  "payload": {
//	    "sub": "1234567890",
//	    "iat": 1516239022
//	  },
//	  "signature": "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
//	}
type Token struct {
	// Header contains the decoded JOSE (JSON Object Signing and Encryption)
	// header. Typically includes "alg" (algorithm) and "typ" (type) fields.
	Header any `json:"header"`

	// Payload contains the decoded claims. This may include registered
	// claims (iss, sub, aud, exp, etc.) and custom application claims.
	Payload any `json:"payload"`

	// Signature is the raw base64url-encoded signature string.
	// It is not decoded as it's binary data, not JSON.
	Signature string `json:"signature"`
}
