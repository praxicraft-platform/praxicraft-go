package praxicraft

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// VerifySignature verifies an X-Praxicraft-Signature header.
//
// Assess signs the raw request body with HMAC-SHA256 using the webhook
// secret (whsec_…). Canonical header value is sha256=<hex>.
// Legacy raw-hex signatures are also accepted.
//
// A nil body is treated as an empty payload (same as []byte{}), so callers
// can pass either for empty POST bodies. Returns false for missing secret or
// signature. Never panics on attacker-controlled signature strings.
func VerifySignature(secret string, body []byte, headerSig string) bool {
	if secret == "" || headerSig == "" {
		return false
	}
	if body == nil {
		body = []byte{}
	}

	expected := signBody(secret, body)
	if len(headerSig) >= 7 && headerSig[:7] == "sha256=" {
		return hmac.Equal([]byte(headerSig), []byte(expected))
	}
	legacy := expected[len("sha256="):]
	return hmac.Equal([]byte(headerSig), []byte(legacy)) || hmac.Equal([]byte(headerSig), []byte(expected))
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
