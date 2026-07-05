package connect

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeOAuth struct {
	exchangeCodeErr       error
	exchangeCodeResult    igapi.ShortLivedToken
	exchangeLongLivedErr  error
	exchangeLongLivedResp igapi.LongLivedToken
}

func (f *fakeOAuth) ExchangeCode(_ context.Context, _ string) (igapi.ShortLivedToken, error) {
	if f.exchangeCodeErr != nil {
		return igapi.ShortLivedToken{}, f.exchangeCodeErr
	}
	return f.exchangeCodeResult, nil
}

func (f *fakeOAuth) ExchangeLongLived(_ context.Context, _ string) (igapi.LongLivedToken, error) {
	if f.exchangeLongLivedErr != nil {
		return igapi.LongLivedToken{}, f.exchangeLongLivedErr
	}
	return f.exchangeLongLivedResp, nil
}

type fakeIdentity struct {
	err    error
	result igapi.MeResult
}

func (f *fakeIdentity) Me(_ context.Context) (igapi.MeResult, error) {
	if f.err != nil {
		return igapi.MeResult{}, f.err
	}
	return f.result, nil
}

type fakeStore struct {
	err      error
	captured UpsertAccountParams
	called   bool
}

func (f *fakeStore) UpsertAccount(_ context.Context, p UpsertAccountParams) (dbgen.Account, error) {
	f.called = true
	f.captured = p
	if f.err != nil {
		return dbgen.Account{}, f.err
	}
	return dbgen.Account{IgUserID: p.IgUserID}, nil
}

// newTestHandler builds a Handler wired with fakes, bypassing New's production wiring.
func newTestHandler(appSecret string, oauth oauthExchanger, identity identityResolver, store accountUpserter) *Handler {
	return &Handler{
		oauth: oauth,
		authorize: func(state string, scopes []string) string {
			return "https://www.instagram.com/oauth/authorize?state=" + state
		},
		newClient: func(_ string) identityResolver { return identity },
		appSecret: appSecret,
		store:     store,
		log:       testLogger(),
	}
}

// ── Start ─────────────────────────────────────────────────────────────────────

func TestStart_RedirectsWithState(t *testing.T) {
	h := newTestHandler("secret", &fakeOAuth{}, &fakeIdentity{}, &fakeStore{})

	req := httptest.NewRequest(http.MethodGet, "/connect/instagram", nil)
	w := httptest.NewRecorder()
	h.Start(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header to be set")
	}
}

// ── Callback ──────────────────────────────────────────────────────────────────

func validState(t *testing.T, secret string) string {
	t.Helper()
	s, err := NewState(secret)
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	return s
}

func TestCallback_Success(t *testing.T) {
	oauth := &fakeOAuth{
		exchangeCodeResult:    igapi.ShortLivedToken{AccessToken: "short-tok", UserID: "1"},
		exchangeLongLivedResp: igapi.LongLivedToken{AccessToken: "long-tok", TokenType: "bearer", ExpiresIn: 5184000},
	}
	identity := &fakeIdentity{result: igapi.MeResult{UserID: "17841400", Username: "olshop_budi", AccountType: "BUSINESS"}}
	store := &fakeStore{}
	h := newTestHandler("secret", oauth, identity, store)

	state := validState(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?code=abc&state="+state, nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !store.called {
		t.Fatal("expected store.UpsertAccount to be called")
	}
	if store.captured.IgUserID != "17841400" {
		t.Errorf("expected IgUserID=17841400, got %q", store.captured.IgUserID)
	}
	if store.captured.AccessToken != "long-tok" {
		t.Errorf("expected AccessToken=long-tok, got %q", store.captured.AccessToken)
	}
}

func TestCallback_InvalidState_Returns403(t *testing.T) {
	h := newTestHandler("secret", &fakeOAuth{}, &fakeIdentity{}, &fakeStore{})

	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?code=abc&state=garbage", nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCallback_MissingCode_Returns400(t *testing.T) {
	h := newTestHandler("secret", &fakeOAuth{}, &fakeIdentity{}, &fakeStore{})

	state := validState(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?state="+state, nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCallback_OAuthDenied_Returns400(t *testing.T) {
	h := newTestHandler("secret", &fakeOAuth{}, &fakeIdentity{}, &fakeStore{})

	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?error=access_denied", nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCallback_ExchangeCodeFails_Returns502(t *testing.T) {
	oauth := &fakeOAuth{exchangeCodeErr: errors.New("igapi: ExchangeCode: boom")}
	h := newTestHandler("secret", oauth, &fakeIdentity{}, &fakeStore{})

	state := validState(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?code=abc&state="+state, nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestCallback_ExchangeLongLivedFails_Returns502(t *testing.T) {
	oauth := &fakeOAuth{
		exchangeCodeResult:   igapi.ShortLivedToken{AccessToken: "short-tok"},
		exchangeLongLivedErr: errors.New("igapi: ExchangeLongLived: boom"),
	}
	h := newTestHandler("secret", oauth, &fakeIdentity{}, &fakeStore{})

	state := validState(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?code=abc&state="+state, nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestCallback_MeFails_Returns502(t *testing.T) {
	oauth := &fakeOAuth{
		exchangeCodeResult:    igapi.ShortLivedToken{AccessToken: "short-tok"},
		exchangeLongLivedResp: igapi.LongLivedToken{AccessToken: "long-tok"},
	}
	identity := &fakeIdentity{err: errors.New("igapi: Me: boom")}
	h := newTestHandler("secret", oauth, identity, &fakeStore{})

	state := validState(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?code=abc&state="+state, nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestCallback_StoreFails_Returns500(t *testing.T) {
	oauth := &fakeOAuth{
		exchangeCodeResult:    igapi.ShortLivedToken{AccessToken: "short-tok"},
		exchangeLongLivedResp: igapi.LongLivedToken{AccessToken: "long-tok"},
	}
	identity := &fakeIdentity{result: igapi.MeResult{UserID: "1", Username: "u"}}
	store := &fakeStore{err: errors.New("db: boom")}
	h := newTestHandler("secret", oauth, identity, store)

	state := validState(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/connect/instagram/callback?code=abc&state="+state, nil)
	w := httptest.NewRecorder()
	h.Callback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
