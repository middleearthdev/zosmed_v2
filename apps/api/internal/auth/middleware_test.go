package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// echoHandler is a next-handler stub that reports whether RequireUser
// injected a user into the context, and writes 200 when reached at all —
// proving whether the middleware called next.ServeHTTP.
func echoHandler(t *testing.T, reached *bool, gotUser *UserDTO) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		if u, ok := UserFromContext(r.Context()); ok {
			*gotUser = u
		}
		w.WriteHeader(http.StatusOK)
	}
}

func TestRequireUser_ValidCookie_CallsNextWithUser(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "mw-valid@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	rawToken := "raw-valid-token"
	if err := store.CreateSession(ctx, hashToken(rawToken), u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var reached bool
	var gotUser UserDTO
	mw := RequireUser(store)
	handler := mw(echoHandler(t, &reached, &gotUser))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: rawToken})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !reached {
		t.Fatal("expected next handler to be called")
	}
	if gotUser.Email != "mw-valid@example.com" {
		t.Errorf("expected injected user email to match, got %q", gotUser.Email)
	}
}

func TestRequireUser_MissingCookie_Returns401(t *testing.T) {
	store, _ := newTestStore()
	var reached bool
	var gotUser UserDTO
	mw := RequireUser(store)
	handler := mw(echoHandler(t, &reached, &gotUser))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil) // no cookie
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if reached {
		t.Error("expected next handler NOT to be called")
	}
}

func TestRequireUser_InvalidCookie_Returns401(t *testing.T) {
	store, _ := newTestStore()
	var reached bool
	var gotUser UserDTO
	mw := RequireUser(store)
	handler := mw(echoHandler(t, &reached, &gotUser))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "some-token-that-was-never-issued"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if reached {
		t.Error("expected next handler NOT to be called")
	}
}

func TestRequireUser_ExpiredSession_Returns401(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "mw-expired@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	rawToken := "raw-expired-token"
	if err := store.CreateSession(ctx, hashToken(rawToken), u.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var reached bool
	var gotUser UserDTO
	mw := RequireUser(store)
	handler := mw(echoHandler(t, &reached, &gotUser))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: rawToken})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for an expired session, got %d", w.Code)
	}
	if reached {
		t.Error("expected next handler NOT to be called")
	}
}
