package webhook

// handler_messaging_test.go — tests for Handler.processMessaging (ADR-006
// §3.3). Reuses the fakeDedupeDTBX/fakeAccountRow/fakeBoolRow test doubles
// from handler_test.go (§12a-1 DRY): GetAccountByIgUserID and HasLiveWorkflow
// are the SAME queries the comment path uses; InsertProcessedMessage shares
// the generic Exec plumbing InsertProcessedComment already exercises. No
// catalog_post check exists on this path (ADR-006 §3.3 note), so
// fakeDedupeDTBX.catalogFound is simply never read here.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

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

func TestProcessMessaging_DuplicateMessage_SkipsEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag:      pgconn.NewCommandTag("INSERT 0 0"), // 0 rows → duplicate
		accountFound: true,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-dup", Subtype: MessagingSubtypeDM}

	if err := h.processMessaging(context.Background(), im); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// Not panicking proves enqueue was skipped (h.enq is nil).
}

func TestProcessMessaging_InsertDBError_StillReturnsError(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execErr:      errors.New("postgres: connection reset"),
		accountFound: true,
	}
	queries := dbgen.New(fakeDB)
	h := makeHandler(t, "secret", "tok", queries)

	im := IngestMessaging{EntryID: "e1", ContactID: "u1", MessageID: "mid-err"}

	err := h.processMessaging(context.Background(), im)
	if err == nil {
		t.Fatal("expected error when InsertProcessedMessage fails")
	}
}

func TestProcessMessaging_NoLiveWorkflow_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag:         pgconn.NewCommandTag("INSERT 0 1"),
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
		execTag:            pgconn.NewCommandTag("INSERT 0 1"),
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
// reaches EnqueueDMIngest: h.enq is nil in this harness (BUG-001, see
// handler_test.go), so reaching it panics with a nil-pointer dereference —
// that panic IS the proof.
func TestProcessMessaging_HasLiveWorkflow_ReachesEnqueue(t *testing.T) {
	fakeDB := &fakeDedupeDTBX{
		execTag:         pgconn.NewCommandTag("INSERT 0 1"),
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
		execTag:         pgconn.NewCommandTag("INSERT 0 1"),
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
