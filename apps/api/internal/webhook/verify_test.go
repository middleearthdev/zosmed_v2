package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// makeHeader builds a "sha256=<hex>" header value for the given body+secret.
func makeHeader(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// ── VerifySignature ───────────────────────────────────────────────────────────

func TestVerifySignature_Valid(t *testing.T) {
	body := []byte(`{"object":"instagram","entry":[]}`)
	secret := "supersecret"
	header := makeHeader(body, secret)

	if !VerifySignature(body, header, secret) {
		t.Fatal("expected VerifySignature to return true for valid signature")
	}
}

func TestVerifySignature_TamperedBody(t *testing.T) {
	body := []byte(`{"object":"instagram","entry":[]}`)
	secret := "supersecret"
	header := makeHeader(body, secret)

	tampered := []byte(`{"object":"instagram","entry":[],"injected":true}`)
	if VerifySignature(tampered, header, secret) {
		t.Fatal("expected VerifySignature to return false for tampered body")
	}
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	body := []byte(`{"object":"instagram"}`)
	header := makeHeader(body, "correct-secret")

	if VerifySignature(body, header, "wrong-secret") {
		t.Fatal("expected VerifySignature to return false when secret is wrong")
	}
}

func TestVerifySignature_MissingPrefix(t *testing.T) {
	body := []byte(`{"object":"instagram"}`)
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write(body)
	// header without "sha256=" prefix
	header := hex.EncodeToString(mac.Sum(nil))

	if VerifySignature(body, header, "s") {
		t.Fatal("expected false when header lacks sha256= prefix")
	}
}

func TestVerifySignature_InvalidHex(t *testing.T) {
	if VerifySignature([]byte("body"), "sha256=ZZZNOTVALIDHEX", "secret") {
		t.Fatal("expected false for invalid hex in header")
	}
}

func TestVerifySignature_EmptyHeader(t *testing.T) {
	if VerifySignature([]byte("body"), "", "secret") {
		t.Fatal("expected false for empty header")
	}
}

// ── VerifyChallenge ───────────────────────────────────────────────────────────

func TestVerifyChallenge_Valid(t *testing.T) {
	c, ok := VerifyChallenge("my-token", "subscribe", "my-token", "abc123")
	if !ok {
		t.Fatal("expected ok=true for valid subscribe challenge")
	}
	if c != "abc123" {
		t.Fatalf("expected challenge %q, got %q", "abc123", c)
	}
}

func TestVerifyChallenge_WrongMode(t *testing.T) {
	_, ok := VerifyChallenge("tok", "unsubscribe", "tok", "ch")
	if ok {
		t.Fatal("expected ok=false for mode != 'subscribe'")
	}
}

func TestVerifyChallenge_WrongToken(t *testing.T) {
	_, ok := VerifyChallenge("correct", "subscribe", "wrong", "ch")
	if ok {
		t.Fatal("expected ok=false when token does not match verifyToken")
	}
}

func TestVerifyChallenge_EmptyChallenge(t *testing.T) {
	// Meta may send an empty challenge in edge cases; we still echo it.
	c, ok := VerifyChallenge("t", "subscribe", "t", "")
	if !ok {
		t.Fatal("expected ok=true even with empty challenge")
	}
	if c != "" {
		t.Fatalf("expected empty challenge echoed, got %q", c)
	}
}
