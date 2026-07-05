package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// futureExpiry returns a session expiry far enough in the future for tests
// that don't care about TTL precision.
func futureExpiry() time.Time {
	return time.Now().Add(time.Hour)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestHandler() (*Handler, *Store, *fakeAuthDBTX) {
	store, db := newTestStore()
	return New(store, false, testLogger()), store, db
}

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, body []byte) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, body)
	}
	return e
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestRegister_Success_SetsCookieAndReturnsUser(t *testing.T) {
	h, _, _ := newTestHandler()

	body := `{"email":"New@Example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cookieName {
		t.Fatalf("expected a %q cookie to be set, got %v", cookieName, cookies)
	}

	env := decodeEnvelope(t, w.Body.Bytes())
	var got MeResponse
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if got.User.Email != "new@example.com" {
		t.Errorf("expected email to be normalised to lower-case, got %q", got.User.Email)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"onboardingCompleted"`)) {
		t.Error("expected response to include onboardingCompleted")
	}
	if bytes.Contains(w.Body.Bytes(), []byte("password")) {
		t.Error("response must never contain password/password_hash")
	}
}

func TestRegister_DuplicateEmail_Returns409(t *testing.T) {
	h, _, _ := newTestHandler()

	body := `{"email":"dup@example.com","password":"password123"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	h.Register(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	h.Register(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w2.Code)
	}
	env := decodeEnvelope(t, w2.Body.Bytes())
	if env.Error == nil || env.Error.Code != "email_taken" {
		t.Errorf("expected error code email_taken, got %+v", env.Error)
	}
}

func TestRegister_ShortPassword_Returns400(t *testing.T) {
	h, _, _ := newTestHandler()
	body := `{"email":"short@example.com","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegister_TooLongPassword_Returns400(t *testing.T) {
	h, _, _ := newTestHandler()
	// 73 bytes — bcrypt would reject >72; handler must return a clean 400, not 500.
	long := strings.Repeat("a", MaxPasswordLen+1)
	body := `{"email":"long@example.com","password":"` + long + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for >72-byte password, got %d", w.Code)
	}
}

func TestRegister_InvalidEmail_Returns400(t *testing.T) {
	h, _, _ := newTestHandler()
	body := `{"email":"not-an-email","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegister_WithSegment_PersistsSegment(t *testing.T) {
	h, _, _ := newTestHandler()
	body := `{"email":"seller@example.com","password":"password123","segment":"seller"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	var got MeResponse
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if got.User.Segment == nil || *got.User.Segment != "seller" {
		t.Errorf("expected segment=seller, got %v", got.User.Segment)
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestLogin_Success_SetsCookie(t *testing.T) {
	h, store, _ := newTestHandler()
	ctx := context.Background()
	hash, _ := HashPassword("correct-password")
	if _, err := store.CreateUser(ctx, "login@example.com", hash); err != nil {
		t.Fatalf("seed CreateUser: %v", err)
	}

	body := `{"email":"login@example.com","password":"correct-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cookieName {
		t.Fatalf("expected a %q cookie, got %v", cookieName, cookies)
	}
}

func TestLogin_WrongPassword_Returns401(t *testing.T) {
	h, store, _ := newTestHandler()
	ctx := context.Background()
	hash, _ := HashPassword("correct-password")
	if _, err := store.CreateUser(ctx, "wrongpw@example.com", hash); err != nil {
		t.Fatalf("seed CreateUser: %v", err)
	}

	body := `{"email":"wrongpw@example.com","password":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "invalid_credentials" {
		t.Errorf("expected error code invalid_credentials, got %+v", env.Error)
	}
}

func TestLogin_UnknownEmail_Returns401(t *testing.T) {
	h, _, _ := newTestHandler()
	body := `{"email":"ghost@example.com","password":"whatever123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "invalid_credentials" {
		t.Errorf("expected error code invalid_credentials (must not leak 'no such user'), got %+v", env.Error)
	}
}

func TestLogin_ReturnsLinkedAccount(t *testing.T) {
	h, store, db := newTestHandler()
	ctx := context.Background()
	hash, _ := HashPassword("correct-password")
	u, err := store.CreateUser(ctx, "withaccount@example.com", hash)
	if err != nil {
		t.Fatalf("seed CreateUser: %v", err)
	}
	db.putAccount(u.ID, dbgen.Account{
		IgUserID: "17841400", Handle: "olshop.aurora", DisplayName: "Aurora Olshop", Status: "connected",
	})

	body := `{"email":"withaccount@example.com","password":"correct-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	env := decodeEnvelope(t, w.Body.Bytes())
	var got MeResponse
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if got.Account == nil || got.Account.Handle != "olshop.aurora" {
		t.Errorf("expected linked account in login response, got %+v", got.Account)
	}
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestLogout_RevokesSessionAndClearsCookie(t *testing.T) {
	h, store, _ := newTestHandler()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "logout2@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	rawToken := "raw-logout-token"
	if err := store.CreateSession(ctx, hashToken(rawToken), u.ID, futureExpiry()); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: rawToken})
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected the cookie to be cleared (negative MaxAge), got %v", cookies)
	}

	if _, err := store.SessionUser(ctx, hashToken(rawToken)); err == nil {
		t.Error("expected session to be revoked after Logout")
	}
}

func TestLogout_NoCookie_NoOp200(t *testing.T) {
	h, _, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 even with no cookie, got %d", w.Code)
	}
}

// ── Me ────────────────────────────────────────────────────────────────────────

func TestMe_ReturnsUserFromContext(t *testing.T) {
	h, _, _ := newTestHandler()

	userDTO := UserDTO{ID: "00000000-0000-0000-0000-000000000001", Email: "me@example.com", OnboardingCompleted: false}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = req.WithContext(WithUser(req.Context(), userDTO))
	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("me@example.com")) {
		t.Error("expected response to contain the context user's email")
	}
}
