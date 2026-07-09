package webhook

// handler_messaging_test.go — tests for Handler.processMessaging (ADR-006
// §3.3). Reuses the fakeDedupeDTBX/fakeAccountRow/fakeBoolRow/fakeEnqueuer
// test doubles from handler_test.go (§12a-1 DRY): GetAccountByIgUserID and
// HasLiveWorkflow are the SAME queries the comment path uses;
// InsertProcessedMessage shares the generic Exec plumbing InsertProcessedComment
// already exercises, and ExistsProcessedMessage shares fakeDedupeDTBX's
// ExistsProcessed* dispatch branch (ADR-007 §2.2). No catalog_post check
// exists on this path (ADR-006 §3.3 note), so fakeDedupeDTBX.catalogFound is
// simply never read here.

import (
	"context"
	"errors"
	"testing"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// TestProcessMessaging_UnknownAccount_SkipsSafely mirrors AC-9 for the
// messaging surface: an unresolvable entry.id must skip safely (never panic,
// never reach enqueue) since h.enq is nil in this harness (BUG-001).
func TestProcessMessaging_UnknownAccount_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{accountFound: false}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{
		EntryID:   "unknown-igsid",
		ContactID: "u1",
		MessageID: "mid-1",
		Subtype:   MessagingSubtypeDM,
	}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error for unknown account, got %v", err)
	}
	// Not panicking proves enqueue was never reached (h.enq is nil).
}

// TestProcessMessaging_EchoSelfEvent_SkipsSafely is a regression for the review
// finding that a messaging event whose sender IS the account (ContactID ==
// EntryID) — the account's own outbound DM echoed/synced back — must skip
// BEFORE account resolution/dedupe/enqueue, so it can never self-trigger a
// [dm-received → send-dm] loop. Not panicking proves the nil enqueuer was never
// reached (guard returns before any DB call).
func TestProcessMessaging_EchoSelfEvent_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{accountFound: true}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{
		EntryID:   "igsid-account-1",
		ContactID: "igsid-account-1", // sender == account → echo/self
		MessageID: "mid-echo",
		Subtype:   MessagingSubtypeDM,
	}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error for echo/self event, got %v", err)
	}
}

func TestProcessMessaging_RealDBError_NotSwallowed(t *testing.T) {
	dbErr := errors.New("connection refused")
	fakeDB := &fakeDedupeDTBX{accountScanErr: dbErr}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-1"}

	err := h.processMessaging(context.Background(), im)
	if err == nil {
		t.Fatal("expected error on real DB failure, got nil")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped DB error, got %v", err)
	}
}

// TestProcessMessaging_DuplicateMessage_SkipsEnqueue verifies the ingest-layer
// dedupe read-check (ADR-007 §2.2 step 2): when ExistsProcessedMessage
// reports true, the message is silently skipped and EnqueueDMIngest is NOT
// called.
func TestProcessMessaging_DuplicateMessage_SkipsEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true,
		existsProcessed: true, // already processed → skip before enqueue
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-dup", Subtype: MessagingSubtypeDM}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Not panicking proves enqueue was skipped (h.enq is nil).
}

// TestProcessMessaging_InsertDBError_NotEscalated is the DM mirror of
// scenario 7 (ADR-007 §5): enqueue succeeds but the confirmation ledger
// write (InsertProcessedMessage) fails — processMessaging must NOT escalate
// that into an error (the task is already durably enqueued).
func TestProcessMessaging_InsertDBError_NotEscalated(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true,
		hasLiveWorkflow: true,
		execErr:         errors.New("postgres: connection reset"),
	}
	queries := dbgen.New(fakeDB)
	enq := &fakeEnqueuer{}
	h := makeHandlerWithEnq(t, queries, enq)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-err", Subtype: MessagingSubtypeDM}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error even though the ledger write failed, got %v", err)
	}
	if enq.dmCalls != 1 {
		t.Errorf("expected 1 enqueue attempt, got %d", enq.dmCalls)
	}
	if fakeDB.execCalls != 1 {
		t.Errorf("expected 1 ledger write attempt, got %d", fakeDB.execCalls)
	}
}

// TestProcessMessaging_EnqueueSucceedsLedgerFails_RedeliverConverges is the DM
// mirror of TestProcessComment_EnqueueSucceedsLedgerFails_RedeliverConverges
// (scenario 7, ADR-007 §5) — TestProcessMessaging_InsertDBError_NotEscalated
// above only proves the FIRST delivery doesn't escalate; this test also
// drives the re-delivery to convergence: a second processMessaging call for
// the identical event enqueues again (idempotent at the enqueue.Client layer
// via asynq.TaskID — unit-tested separately in enqueue_test.go) and the
// ledger write now succeeds.
func TestProcessMessaging_EnqueueSucceedsLedgerFails_RedeliverConverges(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true,
		hasLiveWorkflow: true,
		execErr:         errors.New("postgres: connection reset"), // ledger write fails first time
	}
	queries := dbgen.New(fakeDB)
	enq := &fakeEnqueuer{}
	h := makeHandlerWithEnq(t, queries, enq)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-ledgerfail", Subtype: MessagingSubtypeDM}

	// First delivery: enqueue succeeds, ledger write fails — must NOT propagate.
	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error even though the ledger write failed, got %v", err)
	}
	if enq.dmCalls != 1 {
		t.Errorf("expected 1 enqueue attempt, got %d", enq.dmCalls)
	}
	if fakeDB.execCalls != 1 {
		t.Errorf("expected 1 ledger write attempt, got %d", fakeDB.execCalls)
	}

	// Re-deliver the same event: ExistsProcessedMessage is still false (the
	// ledger never landed), so the handler enqueues again and writes the
	// ledger successfully this time.
	fakeDB.execErr = nil
	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected the re-delivery to converge, got %v", err)
	}
	if enq.dmCalls != 2 {
		t.Errorf("expected 2 enqueue attempts total (idempotent via asynq.TaskID — no second task), got %d", enq.dmCalls)
	}
	if fakeDB.execCalls != 2 {
		t.Errorf("expected 2 ledger write attempts total, got %d", fakeDB.execCalls)
	}
}

func TestProcessMessaging_NoLiveWorkflow_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true,
		hasLiveWorkflow: false,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-1", Subtype: MessagingSubtypeDM}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Not panicking proves enqueue was skipped (no live workflow, no catalog
	// pre-screen exists on this path — ADR-006 §3.3).
}

func TestProcessMessaging_HasLiveWorkflowDBError_NotSwallowed(t *testing.T) {
	dbErr := errors.New("redis-like transient failure")
	fakeDB := &fakeDedupeDTBX{
		accountFound:       true,
		hasLiveWorkflowErr: dbErr,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-1"}

	err := h.processMessaging(context.Background(), im)
	if err == nil {
		t.Fatal("expected error when HasLiveWorkflow fails, got nil")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped DB error, got %v", err)
	}
}

// TestProcessMessaging_HasLiveWorkflow_ReachesEnqueue proves the happy path
// reaches EnqueueDMIngest: h.enq is nil in this harness, so reaching it
// panics with a nil-pointer dereference — that panic IS the proof.
func TestProcessMessaging_HasLiveWorkflow_ReachesEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true,
		hasLiveWorkflow: true,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-live", Subtype: MessagingSubtypeDM}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a nil-enqueuer panic proving processMessaging reached EnqueueDMIngest")
		}
	}()
	_ = h.processMessaging(context.Background(), im)
	t.Fatal("expected panic before reaching this line")
}

// ── ADR-007 §5: enqueue-first ordering (Tahap D, DM mirror of scenarios 6/8) ──

// TestProcessMessaging_EnqueueFails_LedgerNotWritten_ThenRetrySucceeds is the
// DM mirror of scenario 6 (ADR-007 §5): EnqueueDMIngest failing must leave
// the processed_message ledger unwritten, so a retry can converge once the
// queue recovers.
func TestProcessMessaging_EnqueueFails_LedgerNotWritten_ThenRetrySucceeds(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{accountFound: true, hasLiveWorkflow: true}
	queries := dbgen.New(fakeDB)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-enqfail", Subtype: MessagingSubtypeDM}

	failingEnq := &fakeEnqueuer{dmErr: errors.New("redis: connection refused")}
	h1 := makeHandlerWithEnq(t, queries, failingEnq)
	if err := h1.processMessaging(context.Background(), im); err == nil {
		t.Fatal("expected error when enqueue fails")
	}
	if fakeDB.execCalls != 0 {
		t.Errorf("ledger must NOT be written when enqueue fails, got %d Exec call(s)", fakeDB.execCalls)
	}

	okEnq := &fakeEnqueuer{}
	h2 := makeHandlerWithEnq(t, queries, okEnq)
	if err := h2.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if okEnq.dmCalls != 1 {
		t.Errorf("expected 1 enqueue attempt on retry, got %d", okEnq.dmCalls)
	}
	if fakeDB.execCalls != 1 {
		t.Errorf("expected ledger written exactly once after the successful retry, got %d", fakeDB.execCalls)
	}
}

// TestProcessMessaging_AlreadyProcessed_SkipsWithoutEnqueue is the DM mirror
// of scenario 8 (ADR-007 §5): an already-processed event must be skipped
// WITHOUT any enqueue attempt.
func TestProcessMessaging_AlreadyProcessed_SkipsWithoutEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{accountFound: true, existsProcessed: true}
	queries := dbgen.New(fakeDB)
	enq := &fakeEnqueuer{}
	h := makeHandlerWithEnq(t, queries, enq)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-already", Subtype: MessagingSubtypeDM}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if enq.dmCalls != 0 {
		t.Errorf("expected no enqueue attempt for an already-processed event, got %d", enq.dmCalls)
	}
}

// ── Incomplete-event guard ────────────────────────────────────────────────────

func TestProcessMessaging_EmptyContactID_EarlyReturn(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil) // nil queries: panics if any DB method is called
	im := IngestMessaging{EntryID: "e1", ContactID: "", MessageID: "mid-1"}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestProcessMessaging_EmptyMessageID_EarlyReturn(t *testing.T) {
	h := makeHandler(t, "secret", "tok", nil)
	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: ""}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── §4b regression guard (messaging surface) ─────────────────────────────────

// TestRegression_ProcessMessaging_NoFollowerOrLiveSemantics guards that the
// messaging ingest path never introduces a §4b capability: it only ever
// resolves accounts/dedupes/gates on the fields ADR-006 defines (ContactID,
// MessageID, EntryID, Subtype) — nothing follower- or IG-Live-shaped.
func TestRegression_ProcessMessaging_NoFollowerOrLiveSemantics(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		accountFound:    true,
		hasLiveWorkflow: false, // skip before enqueue — keep this test enqueue-free
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	for _, subtype := range []string{MessagingSubtypeDM, MessagingSubtypeStoryReply, MessagingSubtypeStoryMention, MessagingSubtypeAdReferral} {
		im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-" + subtype, Subtype: subtype}
		if err := h.processMessaging(context.Background(), im); err != nil {
			t.Fatalf("subtype %q: expected nil error, got %v", subtype, err)
		}
	}
}
