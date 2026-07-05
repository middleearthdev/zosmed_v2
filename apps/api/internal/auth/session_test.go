package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewSessionToken_UniqueAndNonEmpty(t *testing.T) {
	t1, err := newSessionToken()
	if err != nil {
		t.Fatalf("newSessionToken: %v", err)
	}
	t2, err := newSessionToken()
	if err != nil {
		t.Fatalf("newSessionToken: %v", err)
	}
	if t1 == "" || t2 == "" {
		t.Fatal("expected non-empty tokens")
	}
	if t1 == t2 {
		t.Error("expected two generated tokens to differ")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := hashToken("same-token")
	h2 := hashToken("same-token")
	if h1 != h2 {
		t.Error("expected hashToken to be deterministic for the same input")
	}
	if h1 == "same-token" {
		t.Error("hashToken must not return the raw token")
	}
}

func TestHashToken_DifferentInputsDifferentHashes(t *testing.T) {
	if hashToken("token-a") == hashToken("token-b") {
		t.Error("expected different tokens to hash differently")
	}
}

func TestSetSessionCookie_ClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	setSessionCookie(w, "raw-token-value", true)

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != cookieName {
		t.Errorf("expected cookie name %q, got %q", cookieName, c.Name)
	}
	if c.Value != "raw-token-value" {
		t.Errorf("expected cookie value to be the raw token, got %q", c.Value)
	}
	if !c.HttpOnly {
		t.Error("expected HttpOnly=true")
	}
	if !c.Secure {
		t.Error("expected Secure=true when secure=true is passed")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Error("expected SameSite=Lax")
	}

	w2 := httptest.NewRecorder()
	clearSessionCookie(w2, true)
	cleared := w2.Result().Cookies()[0]
	if cleared.MaxAge >= 0 {
		t.Error("expected clearSessionCookie to set a negative MaxAge (expire immediately)")
	}
}

func TestReadSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "abc123"})

	token, ok := readSessionCookie(req)
	if !ok {
		t.Fatal("expected cookie to be found")
	}
	if token != "abc123" {
		t.Errorf("expected token=abc123, got %q", token)
	}
}

func TestReadSessionCookie_Absent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := readSessionCookie(req); ok {
		t.Error("expected ok=false when no cookie is present")
	}
}
