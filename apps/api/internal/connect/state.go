package connect

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// stateTTL is how long a connect-flow state value remains valid. Instagram's
// own authorization code is valid for one hour (ADR-002 §11-R G6); a shorter
// window here is enough for a human to complete the login+consent screen.
const stateTTL = 10 * time.Minute

// NewState creates a signed, opaque anti-CSRF state value for the OAuth
// connect flow (ADR-002 §3.3). No DB/session storage is needed: the value
// itself is self-verifying via HMAC(nonce.timestamp, appSecret) — the same
// App Secret already used for OAuth client_secret and webhook HMAC (DRY §12a-1).
//
// Format: "<nonce-b64>.<unix-ts>.<hmac-b64>"
func NewState(appSecret string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("connect: generate state nonce: %w", err)
	}
	nonceEnc := base64.RawURLEncoding.EncodeToString(nonce)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := nonceEnc + "." + ts
	return payload + "." + sign(payload, appSecret), nil
}

// VerifyState checks the signature and TTL of a state value produced by
// NewState. Returns false on any malformed input, signature mismatch, or
// expiry — the callback handler treats all of these as "reject the callback".
func VerifyState(state, appSecret string) bool {
	parts := strings.SplitN(state, ".", 3)
	if len(parts) != 3 {
		return false
	}
	nonceEnc, tsStr, sig := parts[0], parts[1], parts[2]

	expected := sign(nonceEnc+"."+tsStr, appSecret)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return false
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}
	return time.Since(time.Unix(ts, 0)) <= stateTTL
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
