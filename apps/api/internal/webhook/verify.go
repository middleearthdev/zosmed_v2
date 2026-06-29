// Package webhook handles Meta (Instagram) webhook subscription verification
// and event ingestion for the Zosmed API server.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const signaturePrefix = "sha256="

// VerifySignature checks the X-Hub-Signature-256 header value against
// HMAC-SHA256(body, appSecret). Returns true only when the comparison succeeds.
//
// Uses hmac.Equal for constant-time comparison to prevent timing side-channel
// attacks. Per Meta documentation (ADR-001 §3.2 step 1).
//
// header must have the form "sha256=<lowercase-hex>".
// Returns false on any parse error, missing prefix, or mismatch.
func VerifySignature(body []byte, header, appSecret string) bool {
	if !strings.HasPrefix(header, signaturePrefix) {
		return false
	}
	sigHex := strings.TrimPrefix(header, signaturePrefix)
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(expected, sig)
}

// VerifyChallenge validates a Meta webhook subscription handshake (GET /webhooks/meta).
// Returns (challenge, true) when mode == "subscribe" AND token matches verifyToken.
// Returns ("", false) on any mismatch so the caller can respond with 403.
func VerifyChallenge(verifyToken, mode, token, challenge string) (string, bool) {
	if mode != "subscribe" || token != verifyToken {
		return "", false
	}
	return challenge, true
}
