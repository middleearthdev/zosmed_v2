package webhook

// handler_test.go — HTTP-layer integration tests for Handler.Challenge and Handler.Receive.
//
// Historical note: processComment/processMessaging used to be untestable on
// the "new event → enqueue" path because enqueue.Client was a concrete type
// (wraps *asynq.Client) with no interface extraction — tracked as BUG-001.
// ADR-007 Tahap D closed this by extracting enqueue.Enqueuer (see
// enqueue/enqueue.go); fakeEnqueuer below is the resulting test double, used
// by the enqueue-first ordering tests (ADR-007 §5 scenarios 6–8).

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
	"github.com/zosmed/zosmed/libs/platform/tasks"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// makeHandler creates a Handler with the given appSecret, verifyToken and
// optional queries. enq is always nil here — every test using this helper
// deliberately never exercises a path that reaches EnqueueCommentIngest/
// EnqueueDMIngest (proven either by a skip returning cleanly, or by a
// nil-enqueuer panic used as reachability proof). Tests that need to observe
// enqueue calls use makeHandlerWithEnq instead.
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

// makeHandlerWithEnq creates a Handler wired with a fake Enqueuer (ADR-007
// Tahap D) so tests can assert enqueue-first ordering: how many times
// EnqueueCommentIngest/EnqueueDMIngest was called, and whether an enqueue
// error propagates without a ledger write.
func makeHandlerWithEnq(t *testing.T, queries *dbgen.Queries, enq *fakeEnqueuer) *Handler {
	t.Helper()
	return &Handler{
		queries:     queries,
		enq:         enq,
		appSecret:   "secret",
		verifyToken: "tok",
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// fakeEnqueuer is a test double for enqueue.Enqueuer. Configurable per-call
// errors + call counters let tests assert the enqueue-first ordering
// contract (ADR-007 §2.2/§3.9) without a real asynq.Client/Redis.
type fakeEnqueuer struct {
	commentErr   error
	dmErr        error
	commentCalls int
	dmCalls      int
}

func (f *fakeEnqueuer) EnqueueCommentIngest(_ context.Context, _ tasks.CommentIngestPayload) error {
	f.commentCalls++
	return f.commentErr
}

func (f *fakeEnqueuer) EnqueueDMIngest(_ context.Context, _ tasks.DMIngestPayload) error {
	f.dmCalls++
	return f.dmErr
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
// B1) path, reordered by ADR-007 §2.2/§3.9 (enqueue-first):
//
//	GetAccountByIgUserID (QueryRow) → ExistsProcessedComment/Message (QueryRow)
//	→ GetActiveCatalogPostByMedia (QueryRow) → HasLiveWorkflow (QueryRow)
//	→ [enqueue, not modelled by this DBTX] → InsertProcessedComment/Message (Exec)
//
// accountFound controls the account-lookup QueryRow leg: true → scans a
// fixed "connected" account row so the flow proceeds; false → pgx.ErrNoRows,
// modelling an unknown IGSID (AC-9: must skip safely, never 500).
// catalogFound / hasLiveWorkflow / hasLiveWorkflowErr control the two legs
// added by ADR-005 §3 B1's OR-based enqueue guard. existsProcessed /
// existsProcessedErr control the ADR-007 dedupe read-check.
//
// QueryRow dispatches on the (stable, sqlc-generated) SQL text — same
// pattern as apps/api/internal/auth/fakedb_test.go — so only the queries a
// given test actually reaches need a configured response; anything else panics.
type fakeDedupeDTBX struct {
	execTag      pgconn.CommandTag
	execErr      error
	execCalls    int // counts InsertProcessedComment/Message calls (ledger writes)
	accountFound bool
	// accountScanErr, when non-nil, is returned by the GetAccountByIgUserID
	// QueryRow scan to model a real DB failure (not ErrNoRows) — see M1.
	accountScanErr error

	// existsProcessed / existsProcessedErr control ExistsProcessedComment/
	// ExistsProcessedMessage (ADR-007 §2.2 step 2 dedupe read-check).
	existsProcessed    bool
	existsProcessedErr error

	// catalogFound controls GetActiveCatalogPostByMedia: true → a row is
	// "found" (Scan succeeds, contents unused by the caller); false →
	// pgx.ErrNoRows (media not registered / not active).
	catalogFound bool
	// hasLiveWorkflow / hasLiveWorkflowErr control HasLiveWorkflow (ADR-005 B1).
	hasLiveWorkflow    bool
	hasLiveWorkflowErr error
}

func (f *fakeDedupeDTBX) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	f.execCalls++
	return f.execTag, f.execErr
}

// Query panics — never reached on the paths under test.
func (f *fakeDedupeDTBX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	panic("fakeDedupeDTBX.Query: unexpected call — wrong code path in handler test")
}

// QueryRow dispatches by SQL text to the account/dedupe/catalog/live-workflow
// lookups exercised by processComment/processMessaging.
func (f *fakeDedupeDTBX) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM account WHERE ig_user_id"):
		return &fakeAccountRow{found: f.accountFound, scanErr: f.accountScanErr}
	case strings.Contains(sql, "FROM processed_comment"), strings.Contains(sql, "FROM processed_message"):
		return &fakeBoolRow{val: f.existsProcessed, err: f.existsProcessedErr}
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

// TestReceive_DuplicateComment_SkipsEnqueue verifies the ingest-layer dedupe
// read-check (ADR-007 §2.2 step 2): when ExistsProcessedComment reports
// true, the comment is silently skipped and EnqueueCommentIngest is NOT
// called.
//
// Proof: h.enq is nil — if EnqueueCommentIngest were called, it would
// nil-dereference and panic, failing the test immediately.
func TestReceive_DuplicateComment_SkipsEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true, // account resolves so we reach the exists-check
		existsProcessed: true, // already processed → skip before enqueue
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

// TestReceive_DBError_StillReturns200 verifies that a DB error in the
// ExistsProcessedComment read-check is logged but the handler still returns
// 200 (preventing Meta from retrying and causing duplicate enqueues — Receive
// never surfaces processComment's error as a non-200 response).
func TestReceive_DBError_StillReturns200(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:       true, // account resolves so we reach the failing exists-check
		existsProcessedErr: errors.New("postgres: connection reset"),
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
// on ordinary comments. h.enq is nil in this test harness, so reaching
// EnqueueCommentIngest panics with a nil-pointer dereference; that panic IS
// the proof (the skip branch above returns cleanly instead).
func TestProcessComment_NotInCatalogButHasLiveWorkflow_ReachesEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
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

// ── ADR-007 §5: enqueue-first ordering (Tahap D, scenarios 6–8) ─────────────
//
// These tests exercise processComment directly (not through Receive/postSigned)
// so they can assert enqueue/ledger call counts precisely — closes BUG-001
// (see file docstring): the enqueuer is now enqueue.Enqueuer, an interface,
// so fakeEnqueuer can observe exactly what processComment does.

// TestProcessComment_EnqueueFails_LedgerNotWritten_ThenRetrySucceeds is
// scenario 6 (ADR-007 §5): when EnqueueCommentIngest fails (e.g. Redis down),
// processComment must return an error WITHOUT writing the processed_comment
// ledger — so a second delivery of the same event (Meta retry, or the
// caller retrying) starts from a clean slate and can succeed once the queue
// recovers.
func TestProcessComment_EnqueueFails_LedgerNotWritten_ThenRetrySucceeds(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound: true,
		catalogFound: true, // short-circuits the live-workflow check
	}
	queries := dbgen.New(fakeDB)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-enqfail-001",
			Text:  "keep C1",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-catalog"},
		},
	}

	// First delivery: enqueue fails (simulated Redis outage).
	failingEnq := &fakeEnqueuer{commentErr: errors.New("redis: connection refused")}
	h1 := makeHandlerWithEnq(t, queries, failingEnq)
	if err := h1.processComment(context.Background(), ic); err == nil {
		t.Fatal("expected error when enqueue fails")
	}
	if failingEnq.commentCalls != 1 {
		t.Errorf("expected 1 enqueue attempt, got %d", failingEnq.commentCalls)
	}
	if fakeDB.execCalls != 0 {
		t.Errorf("ledger must NOT be written when enqueue fails, got %d Exec call(s)", fakeDB.execCalls)
	}

	// Retry (same event, queue has recovered): enqueue now succeeds.
	okEnq := &fakeEnqueuer{}
	h2 := makeHandlerWithEnq(t, queries, okEnq)
	if err := h2.processComment(context.Background(), ic); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if okEnq.commentCalls != 1 {
		t.Errorf("expected 1 enqueue attempt on retry, got %d", okEnq.commentCalls)
	}
	if fakeDB.execCalls != 1 {
		t.Errorf("expected ledger written exactly once after the successful retry, got %d", fakeDB.execCalls)
	}
}

// TestProcessComment_EnqueueSucceedsLedgerFails_RedeliverConverges is
// scenario 7 (ADR-007 §5): enqueue succeeds but the confirmation ledger
// write (InsertProcessedComment) fails — processComment must NOT escalate
// that into an error (the task is already durably enqueued; escalating would
// cause pointless re-processing, and worse, risks a second distinct task if
// the caller's retry logic assumed nothing was enqueued). A re-delivery of
// the same event enqueues again — asynq.TaskID makes that idempotent at the
// enqueue.Client layer (unit-tested in enqueue_test.go), so from this
// handler's point of view it is just "enqueue returns nil again" — and the
// ledger write now succeeds.
func TestProcessComment_EnqueueSucceedsLedgerFails_RedeliverConverges(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound: true,
		catalogFound: true,
		execErr:      errors.New("postgres: connection reset"), // ledger write fails first time
	}
	queries := dbgen.New(fakeDB)
	enq := &fakeEnqueuer{}
	h := makeHandlerWithEnq(t, queries, enq)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-ledgerfail-001",
			Text:  "keep C1",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-catalog"},
		},
	}

	// First delivery: enqueue succeeds, ledger write fails — must NOT propagate.
	if err := h.processComment(context.Background(), ic); err != nil {
		t.Fatalf("expected nil error even though the ledger write failed, got %v", err)
	}
	if enq.commentCalls != 1 {
		t.Errorf("expected 1 enqueue attempt, got %d", enq.commentCalls)
	}
	if fakeDB.execCalls != 1 {
		t.Errorf("expected 1 ledger write attempt, got %d", fakeDB.execCalls)
	}

	// Re-deliver the same event: ExistsProcessedComment is still false (the
	// ledger never landed), so the handler enqueues again and writes the
	// ledger successfully this time.
	fakeDB.execErr = nil
	if err := h.processComment(context.Background(), ic); err != nil {
		t.Fatalf("expected the re-delivery to converge, got %v", err)
	}
	if enq.commentCalls != 2 {
		t.Errorf("expected 2 enqueue attempts total (idempotent via asynq.TaskID — no second task), got %d", enq.commentCalls)
	}
	if fakeDB.execCalls != 2 {
		t.Errorf("expected 2 ledger write attempts total, got %d", fakeDB.execCalls)
	}
}

// TestProcessComment_AlreadyProcessed_SkipsWithoutEnqueue is scenario 8
// (ADR-007 §5): an event whose ledger row already exists (ExistsProcessedComment
// == true) must be skipped WITHOUT any enqueue attempt at all.
func TestProcessComment_AlreadyProcessed_SkipsWithoutEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{accountFound: true, existsProcessed: true}
	queries := dbgen.New(fakeDB)
	enq := &fakeEnqueuer{}
	h := makeHandlerWithEnq(t, queries, enq)

	ic := IngestComment{
		EntryID: "igsid-1",
		Value: CommentValue{
			ID:    "cmt-already-001",
			Text:  "keep C1",
			From:  CommentFrom{ID: "u1", Username: "buyer"},
			Media: CommentMedia{ID: "media-catalog"},
		},
	}

	if err := h.processComment(context.Background(), ic); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if enq.commentCalls != 0 {
		t.Errorf("expected no enqueue attempt for an already-processed event, got %d", enq.commentCalls)
	}
	if fakeDB.execCalls != 0 {
		t.Errorf("expected no ledger write for an already-processed event, got %d", fakeDB.execCalls)
	}
}
