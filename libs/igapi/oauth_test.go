package igapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zosmed/zosmed/libs/igapi"
)

// --- AuthorizeURL (no network) ---

func TestAuthorizeURL_NoScopes_UsesDefaults(t *testing.T) {
	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "s", RedirectURI: "https://app.zosmed.test/callback"}
	got := cfg.AuthorizeURL("state-abc", nil)

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("AuthorizeURL returned invalid URL: %v", err)
	}
	if u.Host != "www.instagram.com" {
		t.Errorf("expected host www.instagram.com, got %q", u.Host)
	}
	if u.Path != "/oauth/authorize" {
		t.Errorf("expected path /oauth/authorize, got %q", u.Path)
	}
	q := u.Query()
	if q.Get("client_id") != "123" {
		t.Errorf("expected client_id=123, got %q", q.Get("client_id"))
	}
	if q.Get("state") != "state-abc" {
		t.Errorf("expected state=state-abc, got %q", q.Get("state"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("expected response_type=code, got %q", q.Get("response_type"))
	}
	for _, scope := range igapi.DefaultScopes {
		if !strings.Contains(q.Get("scope"), scope) {
			t.Errorf("expected default scope %q in scope param, got %q", scope, q.Get("scope"))
		}
	}
}

// --- ExchangeCode ---

func TestExchangeCode_Success_FlatResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"short-tok","user_id":"17841400"}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", srv.URL, "http://unused", "http://unused")
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "sec", RedirectURI: "https://app.zosmed.test/callback"}
	tok, err := cfg.ExchangeCode(context.Background(), "auth-code-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "short-tok" {
		t.Errorf("expected access_token=short-tok, got %q", tok.AccessToken)
	}
	if tok.UserID != "17841400" {
		t.Errorf("expected user_id=17841400, got %q", tok.UserID)
	}
}

func TestExchangeCode_Success_DataArrayResponse(t *testing.T) {
	// Tolerant-parser path: some Meta docs wrap the token in a `data` array.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"access_token":"short-tok-2","user_id":"999"}]}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", srv.URL, "http://unused", "http://unused")
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "sec", RedirectURI: "https://app.zosmed.test/callback"}
	tok, err := cfg.ExchangeCode(context.Background(), "auth-code-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "short-tok-2" {
		t.Errorf("expected access_token=short-tok-2, got %q", tok.AccessToken)
	}
}

func TestExchangeCode_SendsFormParams(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok","user_id":"1"}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", srv.URL, "http://unused", "http://unused")
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "app-1", AppSecret: "secret-1", RedirectURI: "https://cb"}
	_, err := cfg.ExchangeCode(context.Background(), "the-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotForm.Get("client_id") != "app-1" {
		t.Errorf("expected client_id=app-1, got %q", gotForm.Get("client_id"))
	}
	if gotForm.Get("client_secret") != "secret-1" {
		t.Errorf("expected client_secret=secret-1, got %q", gotForm.Get("client_secret"))
	}
	if gotForm.Get("grant_type") != "authorization_code" {
		t.Errorf("expected grant_type=authorization_code, got %q", gotForm.Get("grant_type"))
	}
	if gotForm.Get("code") != "the-code" {
		t.Errorf("expected code=the-code, got %q", gotForm.Get("code"))
	}
}

func TestExchangeCode_GraphError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid code","type":"OAuthException","code":36007}}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", srv.URL, "http://unused", "http://unused")
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "sec", RedirectURI: "https://cb"}
	_, err := cfg.ExchangeCode(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error for invalid code")
	}
	if !strings.Contains(err.Error(), "Invalid code") {
		t.Errorf("expected error to mention Graph error message, got: %v", err)
	}
}

// --- ExchangeLongLived ---

func TestExchangeLongLived_Success(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"long-tok","token_type":"bearer","expires_in":5184000}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", "http://unused", srv.URL, "http://unused")
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "sec", RedirectURI: "https://cb"}
	long, err := cfg.ExchangeLongLived(context.Background(), "short-tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if long.AccessToken != "long-tok" {
		t.Errorf("expected access_token=long-tok, got %q", long.AccessToken)
	}
	if long.ExpiresIn != 5184000 {
		t.Errorf("expected expires_in=5184000, got %d", long.ExpiresIn)
	}
	if gotQuery.Get("grant_type") != "ig_exchange_token" {
		t.Errorf("expected grant_type=ig_exchange_token, got %q", gotQuery.Get("grant_type"))
	}
	if gotQuery.Get("access_token") != "short-tok" {
		t.Errorf("expected access_token param=short-tok, got %q", gotQuery.Get("access_token"))
	}
}

// --- RefreshLongLived ---

func TestRefreshLongLived_Success(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"refreshed-tok","token_type":"bearer","expires_in":5184000}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", "http://unused", "http://unused", srv.URL)
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "sec", RedirectURI: "https://cb"}
	long, err := cfg.RefreshLongLived(context.Background(), "old-long-tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if long.AccessToken != "refreshed-tok" {
		t.Errorf("expected access_token=refreshed-tok, got %q", long.AccessToken)
	}
	if gotQuery.Get("grant_type") != "ig_refresh_token" {
		t.Errorf("expected grant_type=ig_refresh_token, got %q", gotQuery.Get("grant_type"))
	}
}

func TestRefreshLongLived_Failure_MarksCallerResponsible(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`))
	}))
	defer srv.Close()

	restore := igapi.SetOAuthHostsForTest("http://unused", "http://unused", "http://unused", srv.URL)
	defer restore()

	cfg := igapi.OAuthConfig{AppID: "123", AppSecret: "sec", RedirectURI: "https://cb"}
	_, err := cfg.RefreshLongLived(context.Background(), "dead-tok")
	if err == nil {
		t.Fatal("expected error for dead token")
	}
}
