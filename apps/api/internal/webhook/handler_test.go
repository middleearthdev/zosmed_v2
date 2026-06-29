package webhook

// handler_test.go — HTTP-layer integration tests for Handler.Challenge and Handler.Receive.
//
// Design note: Handler.Receive.processComment has a testability gap for the
// "new comment + in-catalog → enqueue" path because enqueue.Client is a concrete
// type (wraps *asynq.Client) with no interface extraction. This is documented as
// BUG-001 in the test suite.  All paths reachable without that constraint are
// covered below.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// makeHandler creates a Handler with the given appSecret, verifyToken and
// optional queries. enq is always nil here because no test below exercises
// a path that reaches EnqueueCommentIngest (see BUG-001 above).
func makeHandler(t *testing.T, appSecret, verifyToken string, queries *dbgen.Queries) *Handler {
	t.Helper()
	return &Handler{
		queries:     queries,
		enq:         nil,
		appSecret:   appSecret,
		verifyToken: verifyToken,
		accountID:   pgtype.UUID{Valid: true}, // 00000000-... in tests
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// postSigned sends a POST to h.Receive with a correctly signed body.
func postSigned(t *testing.T, h *Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/meta", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", makeHeader(body, h.appSecret))
	w := httptest.NewRecorder()
	h.Receive(w, req)
	return w
}

// marshalPayload serialises a MetaPayload for use as a request body.
func marshalPayload(t *testing.T, p MetaPayload) []byte {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshalPayload: %v", err)
	}
	return b
}

// ── fakeDedupeDTBX ───────────────────────────────────────────────────────────

// fakeDedupeDTBX is a minimal dbgen.DBTX stub for testing the dedupe path
// (processComment step 4: InsertProcessedComment → Exec).
//
// Only Exec is exercised on the dedupe path; calling Query or QueryRow would
// mean a different code path was taken, so both panic to surface accidental
// use immediately.
type fakeDedupeDTBX struct {
	execTag pgconn.CommandTag
	execErr error
}

func (f *fakeDedupeDTBX) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return f.execTag, f.execErr
}

// Query panics — never reached on the dedupe/error paths under test.
func (f *fakeDedupeDTBX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	panic("fakeDedupeDTBX.Query: unexpected call — wrong code path in handler test")
}

// QueryRow panics — never reached when Exec returns 0 rows or an error.
func (f *fakeDedupeDTBX) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	panic("fakeDedupeDTBX.QueryRow: unexpected call — wrong code path in handler test")
}

// ── Challenge endpoint ────────────────────────────────────────────────────────

func TestChallenge_Valid(t *testing.T) {
	h := makeHandler(t, "secret", "my-token", nil)
	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/meta?hub.mode=subscribe&hub.verify_token=my-token&hub.challenge=abc123", nil)
	w := httptest.NewRecorder()
	h.Challenge(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "abc123" {
		t.Errorf("expected challenge body %q, got %q", "abc123", got)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type text/plain, got %q", ct)
	}
}

func TestChallenge_WrongToken(t *testing.T) {
	h := makeHandler(t, "secret", "correct-token", nil)
	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/meta?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=xyz", nil)
	w := httptest.NewRecorder()
	h.Challenge(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestChallenge_WrongMode(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)
	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/meta?hub.mode=unsubscribe&hub.verify_token=tok&hub.challenge=xyz", nil)
	w := httptest.NewRecorder()
	h.Challenge(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong mode, got %d", w.Code)
	}
}

func TestChallenge_EmptyParams_Forbidden(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)
	req := httptest.NewRequest(http.MethodGet, "/webhooks/meta", nil)
	w := httptest.NewRecorder()
	h.Challenge(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for missing params, got %d", w.Code)
	}
}

// ── Receive: signature verification ──────────────────────────────────────────

func TestReceive_InvalidSignature_Returns403(t *testing.T) {
	h := makeHandler(t, "real-secret", "tok", nil)
	body := []byte(`{"object":"instagram","entry":[]}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/meta", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=000000deadbeef")
	w := httptest.NewRecorder()
	h.Receive(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid signature, got %d", w.Code)
	}
}

func TestReceive_MissingSignatureHeader_Returns403(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)
	body := []byte(`{"object":"instagram","entry":[]}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/meta", bytes.NewReader(body))
	// No X-Hub-Signature-256 header → VerifySignature rejects empty string
	w := httptest.NewRecorder()
	h.Receive(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for missing signature header, got %d", w.Code)
	}
}

func TestReceive_TamperedBody_Returns403(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)
	originalBody := []byte(`{"object":"instagram","entry":[]}`)

	// Sign original but send tampered body.
	sig := makeHeader(originalBody, "secret")
	tampered := []byte(`{"object":"instagram","entry":[],"injected":true}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/meta", bytes.NewReader(tampered))
	req.Header.Set("X-Hub-Signature-256", sig)
	w := httptest.NewRecorder()
	h.Receive(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for tampered body, got %d", w.Code)
	}
}

// ── Receive: payload processing ───────────────────────────────────────────────

func TestReceive_MalformedJSON_Returns200(t *testing.T) {
	// Meta sends malformed body → handler responds 200 (retrying won't fix it).
	h := makeHandler(t, "secret", "tok", nil)
	body := []byte(`not-valid-json!!!`)

	w := postSigned(t, h, body)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for malformed JSON, got %d", w.Code)
	}
	// Response is the standard envelope: {"data":{"received":true},"error":null}.
	var resp struct {
		Data map[string]bool `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response parse: %v", err)
	}
	if !resp.Data["received"] {
		t.Error("expected received=true in response body")
	}
}

func TestReceive_EmptyEntries_Returns200(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)
	body := marshalPayload(t, MetaPayload{Object: "instagram", Entry: []MetaEntry{}})

	w := postSigned(t, h, body)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestReceive_NonCommentFields_NoDBCall verifies that payloads with only
// "messages" and "mentions" fields skip processComment entirely (no DB access).
// This is also a §4b regression guard: non-comment fields are silently ignored.
func TestReceive_NonCommentFields_NoDBCall(t *testing.T) {
	// nil queries → panics if any DB method is called → would surface immediately.
	h := makeHandler(t, "secret", "tok", nil)

	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Changes: []MetaChange{
				{Field: "messages", Value: json.RawMessage(`{"mid":"msg1"}`)},
				{Field: "mentions", Value: json.RawMessage(`{"id":"mention1"}`)},
			},
		}},
	}
	body := marshalPayload(t, payload)

	w := postSigned(t, h, body)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestReceive_StoryReplyField_NoDBCall verifies that "story_reply" events
// don't flow through the comment-extraction path (separate concern).
func TestReceive_StoryReplyField_NoDBCall(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)

	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID:      "e1",
			Changes: []MetaChange{{Field: "story_reply", Value: json.RawMessage(`{"id":"sr1"}`)}},
		}},
	}
	body := marshalPayload(t, payload)

	w := postSigned(t, h, body)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestReceive_CommentEmptyID_EarlyReturn verifies that a comment with an empty
// ig_comment_id is silently skipped (processComment guard: v.ID == "").
// No DB call happens → nil queries is safe.
func TestReceive_CommentEmptyID_EarlyReturn(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)

	cv := CommentValue{
		ID:    "", // empty → processComment returns nil immediately
		Text:  "keep C1",
		From:  CommentFrom{ID: "u1", Username: "buyer"},
		Media: CommentMedia{ID: "media-001"},
	}
	payload := MetaPayload{
		Object: "instagram",
		Entry:  []MetaEntry{{ID: "e1", Changes: []MetaChange{{Field: "comments", Value: commentValueJSON(t, cv)}}}},
	}
	body := marshalPayload(t, payload)

	w := postSigned(t, h, body)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestReceive_DuplicateComment_SkipsEnqueue verifies the ingest-layer dedupe:
// when InsertProcessedComment returns 0 rows (ON CONFLICT DO NOTHING), the
// comment is silently skipped and EnqueueCommentIngest is NOT called.
//
// Proof: h.enq is nil — if EnqueueCommentIngest were called, h.enq.c would
// nil-dereference and panic, failing the test immediately.
func TestReceive_DuplicateComment_SkipsEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag: pgconn.NewCommandTag("INSERT 0 0"), // 0 rows → duplicate
		execErr: nil,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	cv := CommentValue{
		ID:    "cmt-dup-001",
		Text:  "keep C1",
		From:  CommentFrom{ID: "u1", Username: "buyer"},
		Media: CommentMedia{ID: "media-001"},
	}
	payload := MetaPayload{
		Object: "instagram",
		Entry:  []MetaEntry{{ID: "e1", Changes: []MetaChange{{Field: "comments", Value: commentValueJSON(t, cv)}}}},
	}
	body := marshalPayload(t, payload)

	w := postSigned(t, h, body)

	// Handler ALWAYS returns 200 (Meta would retry on non-200).
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Test not panicking proves enqueue was skipped (h.enq is nil).
}

// TestReceive_DBError_StillReturns200 verifies that a DB error in
// InsertProcessedComment is logged but the handler still returns 200
// (preventing Meta from retrying and causing duplicate enqueues).
func TestReceive_DBError_StillReturns200(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execErr: errors.New("postgres: connection reset"),
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	cv := CommentValue{
		ID:    "cmt-dberr-002",
		Text:  "order",
		From:  CommentFrom{ID: "u2", Username: "shopper"},
		Media: CommentMedia{ID: "media-002"},
	}
	payload := MetaPayload{
		Object: "instagram",
		Entry:  []MetaEntry{{ID: "e1", Changes: []MetaChange{{Field: "comments", Value: commentValueJSON(t, cv)}}}},
	}
	body := marshalPayload(t, payload)

	w := postSigned(t, h, body)

	// Per handler docs: never return non-200 to Meta.
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 even on DB error, got %d", w.Code)
	}
}

// ── §4b Regression guards ─────────────────────────────────────────────────────

// TestRegression_ExtractComments_IgnoresProhibitedFields guards §4b items.
// The webhook handler must NEVER process these field types, regardless of payload content.
func TestRegression_ExtractComments_IgnoresProhibitedFields(t *testing.T) {
	// Each entry maps a webhook field name to the §4b item it would violate.
	prohibitedFields := []struct {
		field  string
		clause string
	}{
		{"new_follower", "§4b.1 — no follower trigger"},
		{"follow", "§4b.1 — no follower trigger"},
		{"live_comments", "§4b.5 — no IG Live comments"},
		{"live_viewers", "§4b.4 — no IG Live viewer count"},
		{"follow_status", "§4b.3 — no follow-status check"},
		{"blast_dm", "§4b.6 — no mass DM blast"},
		{"scrape", "§4b.7 — no scraping"},
	}

	for _, tc := range prohibitedFields {
		t.Run(tc.field, func(t *testing.T) {
			payload := MetaPayload{
				Object: "instagram",
				Entry: []MetaEntry{{
					ID: "e1",
					Changes: []MetaChange{{
						Field: tc.field,
						Value: json.RawMessage(`{"data":"test_value"}`),
					}},
				}},
			}
			comments := ExtractComments(payload)
			if len(comments) != 0 {
				t.Errorf(
					"ExtractComments must NEVER process field %q (%s); got %d comments",
					tc.field, tc.clause, len(comments),
				)
			}
		})
	}
}

// TestRegression_Receive_ProhibitedFields_Returns200NoEnqueue verifies that
// sending a webhook payload with prohibited fields through the full HTTP stack
// returns 200 and makes no DB or enqueue calls.
func TestRegression_Receive_ProhibitedFields_Returns200NoEnqueue(t *testing.T) {
	// nil queries: panics if any DB method is called.
	h := makeHandler(t, "secret", "tok", nil)

	for _, field := range []string{"new_follower", "live_comments", "follow_status"} {
		t.Run(field, func(t *testing.T) {
			payload := MetaPayload{
				Object: "instagram",
				Entry: []MetaEntry{{
					ID:      "e1",
					Changes: []MetaChange{{Field: field, Value: json.RawMessage(`{}`)}},
				}},
			}
			body := marshalPayload(t, payload)

			w := postSigned(t, h, body)

			if w.Code != http.StatusOK {
				t.Errorf("field %q: expected 200, got %d", field, w.Code)
			}
		})
	}
}

// BUG-001 (design gap — reported, not patched):
//
// Handler.processComment contains a path:
//   InsertProcessedComment (Exec) → GetActiveCatalogPostByMedia (QueryRow) → EnqueueCommentIngest
//
// The enqueuer field (h.enq *enqueue.Client) is a concrete type wrapping *asynq.Client.
// There is no interface extraction that would allow injecting a fake enqueuer in tests.
// Consequence: the "new comment in active catalog → enqueue" happy path is not unit-testable
// without either:
//   a) Extracting an Enqueuer interface in enqueue.go (1-line change to prod code), or
//   b) Using a real asynq.Client backed by miniredis (requires adding alicebob/miniredis
//      as a test dependency to apps/api/go.mod).
//
// Additionally, GetActiveCatalogPostByMedia uses QueryRow which returns pgx.Row (concrete
// struct with unexported fields), making it impossible to fake the "catalog not found"
// path without pgxmock.
//
// Recommended fix (low effort): extract Enqueuer interface in enqueue/enqueue.go.
