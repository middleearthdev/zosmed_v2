package igapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zosmed/zosmed/libs/igapi"
)

func TestMe_Success(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Query().Get("fields") != "user_id,username,account_type" {
			t.Errorf("expected fields query param, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user_id":"17841400","username":"olshop_budi","account_type":"BUSINESS"}`))
	}))
	defer srv.Close()

	client := igapi.NewWithBaseURL("tok", srv.URL)
	me, err := client.Me(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if me.UserID != "17841400" {
		t.Errorf("expected user_id=17841400, got %q", me.UserID)
	}
	if me.Username != "olshop_budi" {
		t.Errorf("expected username=olshop_budi, got %q", me.Username)
	}
	if gotPath != "/me" {
		t.Errorf("expected path /me, got %q", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("expected Authorization: Bearer tok, got %q", gotAuth)
	}
}

func TestMe_EmptyUserID_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"username":"no_id"}`))
	}))
	defer srv.Close()

	client := igapi.NewWithBaseURL("tok", srv.URL)
	_, err := client.Me(context.Background())
	if err == nil {
		t.Fatal("expected error when user_id is empty")
	}
}

func TestMe_GraphAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`))
	}))
	defer srv.Close()

	client := igapi.NewWithBaseURL("bad-tok", srv.URL)
	_, err := client.Me(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
