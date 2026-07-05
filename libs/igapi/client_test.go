package igapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zosmed/zosmed/libs/igapi"
)

// newTestServer returns an httptest.Server that responds with the given status and body.
func newTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// --- ReplyToComment ---

func TestReplyToComment_Success(t *testing.T) {
	srv := newTestServer(t, http.StatusOK, `{"id":"123456"}`)
	defer srv.Close()

	client := igapi.NewWithBaseURL("tok", srv.URL)
	if err := client.ReplyToComment(context.Background(), "comment-1", "Halo kak!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplyToComment_GraphAPIError(t *testing.T) {
	body := `{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190,"fbtrace_id":"abc"}}`
	srv := newTestServer(t, http.StatusBadRequest, body)
	defer srv.Close()

	client := igapi.NewWithBaseURL("bad-tok", srv.URL)
	err := client.ReplyToComment(context.Background(), "comment-1", "hi")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "190") {
		t.Errorf("expected error to mention Graph error code 190, got: %v", err)
	}
}

func TestReplyToComment_EmptyCommentID(t *testing.T) {
	client := igapi.New("tok")
	err := client.ReplyToComment(context.Background(), "", "text")
	if err == nil {
		t.Fatal("expected validation error for empty commentID")
	}
}

func TestReplyToComment_EmptyText(t *testing.T) {
	client := igapi.New("tok")
	err := client.ReplyToComment(context.Background(), "c1", "")
	if err == nil {
		t.Fatal("expected validation error for empty text")
	}
}

// --- SendPrivateReply ---

func TestSendPrivateReply_RequestShape(t *testing.T) {
	// Verify the request body has recipient.comment_id set.
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recipient_id":"u1","message_id":"m1"}`))
	}))
	defer srv.Close()

	client := igapi.NewWithBaseURL("tok", srv.URL)
	if err := client.SendPrivateReply(context.Background(), "biz-ig-user", "comment-42", "Hai kak!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recipient, ok := capturedBody["recipient"].(map[string]any)
	if !ok {
		t.Fatalf("expected recipient object in body, got: %v", capturedBody)
	}
	if recipient["comment_id"] != "comment-42" {
		t.Errorf("expected recipient.comment_id=comment-42, got: %v", recipient["comment_id"])
	}
	// id must NOT be set (it is for DM, not private reply)
	if recipient["id"] != nil && recipient["id"] != "" {
		t.Errorf("expected recipient.id to be absent, got: %v", recipient["id"])
	}
}

func TestSendPrivateReply_EmptyInputs(t *testing.T) {
	client := igapi.New("tok")
	cases := []struct {
		name      string
		igUserID  string
		commentID string
		text      string
	}{
		{"empty igUserID", "", "c1", "hi"},
		{"empty commentID", "u1", "", "hi"},
		{"empty text", "u1", "c1", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := client.SendPrivateReply(context.Background(), tc.igUserID, tc.commentID, tc.text)
			if err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

// --- SendDM ---

func TestSendDM_RequestShape(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recipient_id":"u999","message_id":"m999"}`))
	}))
	defer srv.Close()

	client := igapi.NewWithBaseURL("tok", srv.URL)
	if err := client.SendDM(context.Background(), "biz-ig-user", "user-999", "Halo!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recipient, ok := capturedBody["recipient"].(map[string]any)
	if !ok {
		t.Fatalf("expected recipient object, got: %v", capturedBody)
	}
	if recipient["id"] != "user-999" {
		t.Errorf("expected recipient.id=user-999, got: %v", recipient["id"])
	}
	// comment_id must NOT be set
	if cid, ok := recipient["comment_id"].(string); ok && cid != "" {
		t.Errorf("expected recipient.comment_id to be absent, got: %v", cid)
	}
}

func TestSendDM_EmptyInputs(t *testing.T) {
	client := igapi.New("tok")
	cases := []struct{ igUserID, targetUserID, text string }{
		{"", "u", "hi"},
		{"ig", "", "hi"},
		{"ig", "u", ""},
	}
	for _, tc := range cases {
		err := client.SendDM(context.Background(), tc.igUserID, tc.targetUserID, tc.text)
		if err == nil {
			t.Fatalf("expected validation error for input %+v", tc)
		}
	}
}

// --- Auth header ---

func TestClient_SetsAuthorizationHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"x"}`))
	}))
	defer srv.Close()

	client := igapi.NewWithBaseURL("my-token-abc", srv.URL)
	_ = client.ReplyToComment(context.Background(), "c1", "hi")

	if gotAuth != "Bearer my-token-abc" {
		t.Errorf("expected Authorization: Bearer my-token-abc, got: %q", gotAuth)
	}
}
