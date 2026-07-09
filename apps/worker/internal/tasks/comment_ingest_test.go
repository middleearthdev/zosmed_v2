package tasks

// comment_ingest_test.go — unit tests for CommentIngestHandler.ProcessTask,
// mirroring dm_ingest_test.go's scope: account resolution + connected check +
// the catalog/keep-code pre-screen, WITHOUT a fully compiled `live` workflow
// (that branch is exercised at the node level elsewhere — see
// dm_ingest_test.go's file doc comment for the rationale, §12a-1 DRY).
//
// One addition over dm_ingest_test.go: a regression test for ADR-007 §2.3(c)
// / §5 scenario 12 — "RunStore.Insert gagal → task ingest tetap return nil".
// The transitional fallback branch (ADR-004 R3) is the cheapest path to a
// Triggered=true run in this handler (it uses Runner.Engine directly, no
// Loader/Compiler machinery needed), so that's what the regression test
// drives — with a minimal always-matching trigger, not the real seller kit.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/worker/internal/runner"
	"github.com/zosmed/zosmed/apps/worker/internal/wfload"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/workflow"
)

// ── fake DBTX ─────────────────────────────────────────────────────────────────

// fakeScanErrRow implements pgx.Row, always failing (or succeeding with
// nothing scanned, if err is nil) Scan — used where the handler only checks
// err == nil / err != nil and never reads the row's contents back (e.g.
// GetCommentOrderSettings, whose result is discarded on error and the fields
// aren't asserted on in these tests).
type fakeScanErrRow struct{ err error }

func (r *fakeScanErrRow) Scan(_ ...any) error { return r.err }

// fakeCatalogPostRow implements pgx.Row for GetActiveCatalogPostByMedia's
// success case — only ID needs a valid value (comment_ingest.go formats it
// into catalogPostID via uuidx.Format); the rest are zero-value-safe.
type fakeCatalogPostRow struct{}

func (r *fakeCatalogPostRow) Scan(dest ...any) error {
	if p, ok := dest[0].(*pgtype.UUID); ok {
		*p = pgtype.UUID{Bytes: [16]byte{0x02}, Valid: true}
	}
	return nil
}

// fakeCommentIngestDTBX is a minimal dbgen.DBTX stub covering the queries
// CommentIngestHandler.ProcessTask reaches on the fallback-branch path:
// GetAccountByID → GetActiveCatalogPostByMedia → GetCommentOrderSettings →
// ListLiveWorkflowsByAccount (always empty here) → InsertRun.
type fakeCommentIngestDTBX struct {
	account    dbgen.Account
	accountErr error

	catalogFound bool // controls GetActiveCatalogPostByMedia's found/not-found leg

	insertRunErr   error
	insertRunCalls int
}

func (f *fakeCommentIngestDTBX) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	panic("fakeCommentIngestDTBX.Exec: unhandled query: " + sql)
}

func (f *fakeCommentIngestDTBX) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if strings.Contains(sql, "status = 'live'") {
		return emptyRows{}, nil // reuse dm_ingest_test.go's emptyRows (§12a-1)
	}
	panic("fakeCommentIngestDTBX.Query: unhandled query: " + sql)
}

func (f *fakeCommentIngestDTBX) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM account WHERE id"):
		return &fakeAccountByIDRow{account: f.account, err: f.accountErr} // reuse dm_ingest_test.go
	case strings.Contains(sql, "FROM catalog_post"):
		if !f.catalogFound {
			return &fakeScanErrRow{err: pgx.ErrNoRows}
		}
		return &fakeCatalogPostRow{}
	case strings.Contains(sql, "FROM comment_order_settings"):
		// Always "not found" — comment_ingest.go treats any error here as
		// non-fatal and falls back to seller.DefaultHoldSeconds.
		return &fakeScanErrRow{err: pgx.ErrNoRows}
	case strings.Contains(sql, "INSERT INTO workflow_run"):
		// InsertRun is a `:one` query (RETURNING clause) → QueryRow, not Exec.
		f.insertRunCalls++
		return &fakeScanErrRow{err: f.insertRunErr}
	}
	panic("fakeCommentIngestDTBX.QueryRow: unhandled query: " + sql)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func connectedCommentAccount() dbgen.Account {
	return dbgen.Account{
		ID:       pgtype.UUID{Bytes: [16]byte{0x01}, Valid: true},
		IgUserID: "17841400",
		Status:   "connected",
	}
}

func commentIngestTask(t *testing.T, p ptasks.CommentIngestPayload) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return asynq.NewTask(ptasks.TaskCommentIngest, b)
}

func commentBasePayload() ptasks.CommentIngestPayload {
	return ptasks.CommentIngestPayload{
		AccountID:    validAccountID, // shared const, dm_ingest_test.go
		CommentID:    "cmt-1",
		MediaID:      "media-1",
		FromID:       "igsid-user-1",
		FromUsername: "budi",
		Text:         "halo kak",
		CommentAt:    "2026-07-08T10:00:00Z",
	}
}

// alwaysTrigger is a workflow.Trigger stub that matches every event — used to
// force a Triggered=true run through Runner.Engine (the ADR-004 R3 fallback
// branch) without needing the real seller-kit reserve/private-reply nodes
// (which need a wired Sender/Gate to execute). No actions are registered, so
// the run has FilterPassed=false-by-default-empty/no filters and zero action
// steps — Triggered=true is all the regression test below needs.
type alwaysTrigger struct{}

func (alwaysTrigger) Match(_ context.Context, _ workflow.Event) bool { return true }

// alwaysTriggerEngine builds a minimal single-workflow Engine whose one
// trigger always matches — a deliberately trivial substitute for
// runner.CommentToOrderWorkflow in tests that only need Triggered=true.
func alwaysTriggerEngine() *workflow.Engine {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", alwaysTrigger{})
	wf := workflow.WorkflowDef{
		ID:          "test-always-trigger",
		TriggerKeys: []string{"always"},
	}
	return workflow.NewEngine(reg, []workflow.WorkflowDef{wf})
}

// newCommentIngestHandler wires a Runner whose Engine defaults to
// runner.CommentToOrderWorkflow's registry (empty — no seller nodes
// registered) unless eng is provided, letting most tests take the
// "no live workflow and not a keep-code order → skip" early return, and the
// regression test below swap in alwaysTriggerEngine() to reach RunStore.Insert.
func newCommentIngestHandler(fakeDB *fakeCommentIngestDTBX, eng *workflow.Engine) *CommentIngestHandler {
	db := dbgen.New(fakeDB)
	if eng == nil {
		eng = workflow.NewEngine(workflow.NewRegistry(), nil)
	}
	r := &runner.Runner{
		DB:       db,
		Engine:   eng,
		Loader:   wfload.NewLoader(db),
		RunStore: wfload.NewRunStore(db),
	}
	return NewCommentIngestHandler(r, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCommentIngest_UnknownAccount_SkipsSafely(t *testing.T) {
	fakeDB := &fakeCommentIngestDTBX{accountErr: errors.New("no rows in result set")}
	h := newCommentIngestHandler(fakeDB, nil)

	if err := h.ProcessTask(context.Background(), commentIngestTask(t, commentBasePayload())); err != nil {
		t.Fatalf("expected nil error for unknown account, got %v", err)
	}
}

func TestCommentIngest_RealAccountDBError_NotSwallowed(t *testing.T) {
	dbErr := errors.New("connection refused")
	fakeDB := &fakeCommentIngestDTBX{accountErr: dbErr}
	h := newCommentIngestHandler(fakeDB, nil)

	err := h.ProcessTask(context.Background(), commentIngestTask(t, commentBasePayload()))
	if err == nil {
		t.Fatal("expected error on real DB failure, got nil")
	}
}

func TestCommentIngest_AccountNotConnected_Skips(t *testing.T) {
	acc := connectedCommentAccount()
	acc.Status = "expired"
	fakeDB := &fakeCommentIngestDTBX{account: acc}
	h := newCommentIngestHandler(fakeDB, nil)

	if err := h.ProcessTask(context.Background(), commentIngestTask(t, commentBasePayload())); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestCommentIngest_NoLiveWorkflowNoKeepCode_SkipsWithoutError verifies the
// "nothing to do" early return: no `live` workflow AND no keep code on a
// catalog post → skip, never reaching InsertRun.
func TestCommentIngest_NoLiveWorkflowNoKeepCode_SkipsWithoutError(t *testing.T) {
	fakeDB := &fakeCommentIngestDTBX{account: connectedCommentAccount()}
	h := newCommentIngestHandler(fakeDB, nil)

	if err := h.ProcessTask(context.Background(), commentIngestTask(t, commentBasePayload())); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeDB.insertRunCalls != 0 {
		t.Errorf("expected no InsertRun call, got %d", fakeDB.insertRunCalls)
	}
}

func TestCommentIngest_UnmarshalError(t *testing.T) {
	fakeDB := &fakeCommentIngestDTBX{}
	h := newCommentIngestHandler(fakeDB, nil)

	task := asynq.NewTask(ptasks.TaskCommentIngest, []byte("not-json"))
	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("expected error for malformed payload")
	}
}

// TestCommentIngest_FallbackTriggered_RunStoreInsertFails_StillReturnsNil is
// the ADR-007 §2.3(c) / §5 scenario 12 regression: once a workflow has
// Triggered, CommentIngestHandler must NEVER turn a RunStore.Insert failure
// into a retryable error — that would make asynq re-run the whole workflow
// (including any outbound already sent) on the next attempt, violating the
// no-double-send guardrail (§4c). RunStore.Insert failing must be logged and
// swallowed; ProcessTask must still return nil.
func TestCommentIngest_FallbackTriggered_RunStoreInsertFails_StillReturnsNil(t *testing.T) {
	fakeDB := &fakeCommentIngestDTBX{
		account:      connectedCommentAccount(),
		catalogFound: true, // hasCode && inCatalog → fallback branch reachable
		insertRunErr: errors.New("postgres: connection reset"),
	}
	h := newCommentIngestHandler(fakeDB, alwaysTriggerEngine())

	p := commentBasePayload()
	p.Text = "keep C1" // seller.DetectKeepCode must match to take the fallback branch

	if err := h.ProcessTask(context.Background(), commentIngestTask(t, p)); err != nil {
		t.Fatalf("invariant #6c violated: RunStore.Insert failure must not be retried, got error: %v", err)
	}
	if fakeDB.insertRunCalls != 1 {
		t.Fatalf("expected exactly 1 InsertRun attempt, got %d", fakeDB.insertRunCalls)
	}
}

// TestCommentIngest_FallbackTriggered_RunStoreInsertSucceeds is the
// happy-path counterpart: same setup, but InsertRun succeeds — asserts the
// fallback branch actually reaches RunStore.Insert exactly once (proving the
// regression test above is exercising the intended code path, not skipping
// past it).
func TestCommentIngest_FallbackTriggered_RunStoreInsertSucceeds(t *testing.T) {
	fakeDB := &fakeCommentIngestDTBX{
		account:      connectedCommentAccount(),
		catalogFound: true,
	}
	h := newCommentIngestHandler(fakeDB, alwaysTriggerEngine())

	p := commentBasePayload()
	p.Text = "keep C1"

	if err := h.ProcessTask(context.Background(), commentIngestTask(t, p)); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeDB.insertRunCalls != 1 {
		t.Fatalf("expected exactly 1 InsertRun call, got %d", fakeDB.insertRunCalls)
	}
}
