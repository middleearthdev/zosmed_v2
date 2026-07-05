package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// cookieName is the single constant for the session cookie, shared by
// session.go (set/clear) and middleware.go (read) — DRY §12a-1. The FE
// middleware.ts (ADR-003 §8.1) reads the same literal "zsid"; keep both
// in sync if this ever changes.
const cookieName = "zsid"

// sessionTTL is how long a session (and its cookie) remains valid before
// requiring a fresh login (ADR-003 §3).
const sessionTTL = 30 * 24 * time.Hour

// sessionTokenBytes is the size of the random session token before encoding
// (ADR-003 §0 decision 1: "opaque random token (32 byte)").
const sessionTokenBytes = 32

// newSessionToken generates a fresh opaque session token (base64url of 32
// random bytes). The raw token is sent to the browser via cookie; only its
// SHA-256 hash (hashToken) is ever persisted.
func newSessionToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken returns the hex-encoded SHA-256 hash of a raw session token, for
// storage in user_session.token_hash (AC-2: raw token is never persisted).
//
// This is deliberately a plain SHA-256, distinct from connect/state.go's
// HMAC(appSecret) — that value is a stateless, short-lived anti-CSRF nonce;
// this one is a stateful, revocable session lookup key. Different concepts,
// not forced into one DRY abstraction (ADR-003 §12, §12a-4).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// setSessionCookie writes the "zsid" cookie for a freshly created session.
// secure should be true only when APP_ENV=prod (served over HTTPS).
func setSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// clearSessionCookie expires the "zsid" cookie on logout.
func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// readSessionCookie extracts the raw session token from the request, if present.
func readSessionCookie(r *http.Request) (string, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}
