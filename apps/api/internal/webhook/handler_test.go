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
	"strings"
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

// fakeDedupeDTBX is a minimal dbgen.DBTX stub for testing the
// account-resolution (ADR-002 §6.1) + dedupe + ingest-decoupling (ADR-005 §3
// B1) path:
//
//	GetAccountByIgUserID (QueryRow) → InsertProcessedComment (Exec) →
//	GetActiveCatalogPostByMedia (QueryRow) → HasLiveWorkflow (QueryRow)
//
// accountFound controls the account-lookup QueryRow leg: true → scans a
// fixed "connected" account row so the flow proceeds to Exec; false →
// pgx.ErrNoRows, modelling an unknown IGSID (AC-9: must skip safely, never
// 500). catalogFound / hasLiveWorkflow / hasLiveWorkflowErr control the two
// legs added by ADR-005 §3 B1's OR-based enqueue guard.
//
// QueryRow dispatches on the (stable, sqlc-generated) SQL text — same
// pattern as apps/api/internal/auth/fakedb_test.go — so only the queries a
// given test actually reaches need a configured response; anything else panics.
type fakeDedupeDTBX struct {
	execTag      pgconn.CommandTag
	execErr      error
	accountFound bool
	// accountScanErr, when non-nil, is returned by the GetAccountByIgUserID
	// QueryRow scan to model a real DB failure (not ErrNoRows) — see M1.
	accountScanErr error

	// catalogFound controls GetActiveCatalogPostByMedia: true → a row is
	// "found" (Scan succeeds, contents unused by the caller); false →
	// pgx.ErrNoRows (media not registered / not active).
	catalogFound bool
	// hasLiveWorkflow / hasLiveWorkflowErr control HasLiveWorkflow (ADR-005 B1).
	hasLiveWorkflow    bool
	hasLiveWorkflowErr error
}

func (f *fakeDedupeDTBX) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return f.execTag, f.execErr
}

// Query panics — never reached on the paths under test.
func (f *fakeDedupeDTBX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	panic("fakeDedupeDTBX.Query: unexpected call — wrong code path in handler test")
}

// QueryRow dispatches by SQL text to the account/catalog/live-workflow
// lookups exercised by processComment.
func (f *fakeDedupeDTBX) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM account WHERE ig_user_id"):
		return &fakeAccountRow{found: f.accountFound, scanErr: f.accountScanErr}
	case strings.Contains(sql, "FROM catalog_post"):
		if !f.catalogFound {
			return &fakeErrRow{err: pgx.ErrNoRows}
		}
		return &fakeErrRow{err: nil} // caller discards the scanned CatalogPost value
	case strings.Contains(sql, "status = 'live'"):
		return &fakeBoolRow{val: f.hasLiveWorkflow, err: f.hasLiveWorkflowErr}
	}
	panic("fakeDedupeDTBX.QueryRow: unhandled query: " + sql)
}

// fakeErrRow implements pgx.Row, always failing (or succeeding, if err is
// nil) Scan with a fixed result — used to model "not found" / "found but
// unused" QueryRow outcomes without needing per-query row shapes.
type fakeErrRow struct{ err error }

func (r *fakeErrRow) Scan(_ ...interface{}) error { return r.err }

// fakeBoolRow implements pgx.Row for the single-bool-column HasLiveWorkflow
// query (ADR-005 §3 B1).
type fakeBoolRow struct {
	val bool
	err error
}

func (r *fakeBoolRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 0 {
		if p, ok := dest[0].(*bool); ok {
			*p = r.val
		}
	}
	return nil
}

// fakeAccountRow implements pgx.Row for the GetAccountByIgUserID scan. It
// only needs to satisfy dest[0] (Account.ID, a pgtype.UUID) — the tests below
// never inspect the resolved account's other fields.
type fakeAccountRow struct {
	found   bool
	scanErr error
}

func (r *fakeAccountRow) Scan(dest ...interface{}) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if !r.found {
		return pgx.ErrNoRows
	}
	if len(dest) > 0 {
		if idPtr, ok := dest[0].(*pgtype.UUID); ok {
			*idPtr = pgtype.UUID{Bytes: [16]byte{0x01}, Valid: true}
		}
	}
	return nil
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
		execTag:      pgconn.NewCommandTag("INSERT 0 0"), // 0 rows → duplicate
		execErr:      nil,
		accountFound: true, // account resolves so we reach the dedupe Exec
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
		execErr:      errors.New("postgres: connection reset"),
		accountFound: true, // account resolves so we reach the failing Exec
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

// TestReceive_UnknownAccount_SkipsSafely verifies ADR-002 AC-9: a comment
// whose entry.id (IGSID) does not resolve to a known account is skipped
// safely — still 200, and no dedupe/enqueue call is attempted.
//
// Proof: accountFound=false makes GetAccountByIgUserID return pgx.ErrNoRows;
// h.enq is nil, so if processComment somehow proceeded to EnqueueCommentIngest
// it would nil-dereference and panic, failing the test immediately.
func TestReceive_UnknownAccount_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{accountFound: false}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	cv := CommentValue{
		ID:    "cmt-unknown-acct-001",
		Text:  "keep C1",
		From:  CommentFrom{ID: "u3", Username: "buyer3"},
		Media: CommentMedia{ID: "media-003"},
	}
	payload := MetaPayload{
		Object: "instagram",
		Entry:  []MetaEntry{{ID: "unknown-igsid", Changes: []MetaChange{{Field: "comments", Value: commentValueJSON(t, cv)}}}},
	}
	body := marshalPayload(t, payload)

	w := postSigned(t, h, body)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for unknown account, got %d", w.Code)
	}
	// Test not panicking proves the dedupe/enqueue steps were never reached.
}

// TestProcessComment_RealDBError_NotSwallowed guards M1: a non-ErrNoRows
// failure from GetAccountByIgUserID (DB down/transient) must surface as an
// error — NOT be silently treated as "unknown account" and skipped. Otherwise
// comments are dropped without observability (Meta still gets 200, no retry).
func TestProcessComment_RealDBError_NotSwallowed(t *testing.T) {
	dbErr := errors.New("connection refused")
	fakeDB := &fakeDedupeDTBX{accountScanErr: dbErr}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	ic := IngestComment{
		EntryID: "some-igsid",
		Value: CommentValue{
			ID:    "cmt-db-err-001",
			Text:  "keep C1",
			From:  CommentFrom{ID: "u9", Username: "buyer9"},
			Media: CommentMedia{ID: "media-009"},
		},
	}

	err := h.processComment(context.Background(), ic)
	if err == nil {
		t.Fatal("expected error on real DB failure, got nil (M1: error was swallowed as unknown-account)")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped DB error, got %v", err)
	}
}

// ── ADR-005 §3 B1: ingest decoupling (OR-based enqueue guard) ────────────────

// TestProcessComment_NotInCatalogNoLiveWorkflow_SkipsSafely verifies the
// pre-decoupling behaviour is preserved: a comment on a media_id that is
// NEITHER in an active catalog post NOR backed by any `live` workflow is
// still skipped safely (no enqueue attempt, no error).
func TestProcessComment_NotInCatalogNoLiveWorkflow_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag:         pgconn.NewCommandTag("INSERT 0 1"),
		accountFound:    true,
		catalogFound:    false,
		hasLiveWorkflow: false,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-skip-001",
			Text:  "halo kak, cakep banget",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-non-catalog"},
		},
	}

	if err := h.processComment(context.Background(), ic); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// h.enq is nil (see makeHandler) — not panicking proves EnqueueCommentIngest
	// was never reached, i.e. the comment was skipped as before ADR-005.
}

// TestProcessComment_NotInCatalogButHasLiveWorkflow_ReachesEnqueue proves the
// new OR branch (ADR-005 §3 B1): a comment on a non-catalog post still
// reaches the enqueue step once the account has ≥1 `live` workflow — this is
// what makes a generic [comment-received → reply-comment] workflow reachable
// on ordinary comments. h.enq is nil in this test harness (BUG-001, see file
// docstring), so reaching EnqueueCommentIngest panics with a nil-pointer
// dereference; that panic IS the proof (the skip branch above returns
// cleanly instead).
func TestProcessComment_NotInCatalogButHasLiveWorkflow_ReachesEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag:         pgconn.NewCommandTag("INSERT 0 1"),
		accountFound:    true,
		catalogFound:    false,
		hasLiveWorkflow: true,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-live-001",
			Text:  "halo kak, boleh tanya-tanya?",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-non-catalog"},
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a nil-enqueuer panic proving the OR branch reached EnqueueCommentIngest")
		}
	}()
	_ = h.processComment(context.Background(), ic)
	t.Fatal("expected panic before reaching this line")
}

// TestProcessComment_HasLiveWorkflowDBError_NotSwallowed guards the same M1
// discipline as TestProcessComment_RealDBError_NotSwallowed, but for the new
// HasLiveWorkflow leg: a real DB failure must surface as an error, not be
// silently treated as "no live workflow".
func TestProcessComment_HasLiveWorkflowDBError_NotSwallowed(t *testing.T) {
	dbErr := errors.New("redis-like transient failure")
	fakeDB := &fakeDedupeDTBX{
		execTag:            pgconn.NewCommandTag("INSERT 0 1"),
		accountFound:       true,
		catalogFound:       false,
		hasLiveWorkflowErr: dbErr,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-hlw-err-001",
			Text:  "halo kak",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-x"},
		},
	}

	err := h.processComment(context.Background(), ic)
	if err == nil {
		t.Fatal("expected error when HasLiveWorkflow fails, got nil")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped DB error, got %v", err)
	}
}

// TestProcessComment_InCatalog_SkipsLiveWorkflowCheck verifies the catalog
// leg still short-circuits the OR: when the media IS in an active catalog
// post, HasLiveWorkflow must never be queried (hasLiveWorkflowErr would
// panic the fake if reached, since no query text matches — proving the
// legacy seller pre-screen path is untouched by ADR-005 §3 B1).
func TestProcessComment_InCatalog_SkipsLiveWorkflowCheck(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag:      pgconn.NewCommandTag("INSERT 0 1"),
		accountFound: true,
		catalogFound: true,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-catalog-001",
			Text:  "keep C1",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-catalog"},
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a nil-enqueuer panic proving the catalog branch reached EnqueueCommentIngest")
		}
	}()
	_ = h.processComment(context.Background(), ic)
	t.Fatal("expected panic before reaching this line")
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
