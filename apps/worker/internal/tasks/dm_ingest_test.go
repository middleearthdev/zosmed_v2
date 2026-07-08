package tasks

// dm_ingest_test.go — unit tests for DMIngestHandler.ProcessTask (ADR-006
// §4.1), focused on the parts reachable WITHOUT a fully compiled workflow
// (account resolution + connected check + the conversation window upsert,
// which ADR-006 requires to run for EVERY messaging event, R3). The
// engine-run/RunStore branch is exercised at the node level instead
// (trigger_messaging_test.go, filter_conversation_state_test.go,
// action_send_dm_test.go, libs/workflow/nodes) and via
// libs/kits/seller/e2e_test.go's pattern for the comment-ingest sibling —
// mirroring the fact that comment_ingest.go itself has no direct
// DBTX-driven unit test in this codebase either.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/worker/internal/runner"
	"github.com/zosmed/zosmed/apps/worker/internal/wfload"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
)

// ── fake DBTX ─────────────────────────────────────────────────────────────────

// fakeAccountByIDRow implements pgx.Row for GetAccountByID.
type fakeAccountByIDRow struct {
	account dbgen.Account
	err     error
}

func (r *fakeAccountByIDRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*pgtype.UUID) = r.account.ID
	*dest[1].(*string) = r.account.IgUserID
	*dest[2].(*string) = r.account.Handle
	*dest[3].(*string) = r.account.DisplayName
	*dest[4].(*string) = r.account.Status
	*dest[5].(*pgtype.Timestamptz) = r.account.CreatedAt
	*dest[6].(*string) = r.account.AccessToken
	*dest[7].(*string) = r.account.TokenType
	*dest[8].(*[]string) = r.account.Scopes
	*dest[9].(*pgtype.Timestamptz) = r.account.TokenExpiresAt
	*dest[10].(*pgtype.Timestamptz) = r.account.TokenRefreshedAt
	*dest[11].(*pgtype.UUID) = r.account.UserID
	return nil
}

// fakeConversationRow implements pgx.Row for UpsertConversationInteraction.
// Only LastInteractionAt is populated on Scan — the only field dm_ingest.go
// reads back from the upsert result.
type fakeConversationRow struct {
	lastInteractionAt pgtype.Timestamptz
	err               error
}

func (r *fakeConversationRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if p, ok := dest[3].(*pgtype.Timestamptz); ok {
		*p = r.lastInteractionAt
	}
	return nil
}

// emptyRows implements pgx.Rows with zero rows — used for
// ListLiveWorkflowsByAccount so ProcessTask takes the "no live workflow"
// early-return path without needing a fully faked workflow graph.
type emptyRows struct{}

func (emptyRows) Close()                                       {}
func (emptyRows) Err() error                                   { return nil }
func (emptyRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (emptyRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (emptyRows) Next() bool                                   { return false }
func (emptyRows) Scan(...any) error                            { return errors.New("emptyRows: no rows") }
func (emptyRows) Values() ([]any, error)                       { return nil, errors.New("emptyRows: no rows") }
func (emptyRows) RawValues() [][]byte                          { return nil }
func (emptyRows) Conn() *pgx.Conn                              { return nil }

// fakeDMIngestDTBX is a minimal dbgen.DBTX stub covering the queries
// DMIngestHandler.ProcessTask reaches: GetAccountByID → UpsertConversationInteraction
// → ListLiveWorkflowsByAccount (always empty here, keeping the fake small).
type fakeDMIngestDTBX struct {
	account    dbgen.Account
	accountErr error
	upsertErr  error

	upsertCalls int
	lastArgs    []any // captured args of the LAST UpsertConversationInteraction call
}

func (f *fakeDMIngestDTBX) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	panic("fakeDMIngestDTBX.Exec: unexpected call (no live workflow → InsertRun never reached): " + sql)
}

func (f *fakeDMIngestDTBX) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if strings.Contains(sql, "status = 'live'") {
		return emptyRows{}, nil
	}
	panic("fakeDMIngestDTBX.Query: unhandled query: " + sql)
}

func (f *fakeDMIngestDTBX) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM account WHERE id"):
		return &fakeAccountByIDRow{account: f.account, err: f.accountErr}
	case strings.Contains(sql, "INSERT INTO conversation"):
		f.upsertCalls++
		f.lastArgs = args
		lastInteractionAt, _ := args[2].(pgtype.Timestamptz)
		return &fakeConversationRow{lastInteractionAt: lastInteractionAt, err: f.upsertErr}
	}
	panic("fakeDMIngestDTBX.QueryRow: unhandled query: " + sql)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func connectedDMAccount() dbgen.Account {
	return dbgen.Account{
		ID:       pgtype.UUID{Bytes: [16]byte{0x01}, Valid: true},
		IgUserID: "17841400",
		Status:   "connected",
	}
}

func dmIngestTask(t *testing.T, p ptasks.DMIngestPayload) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return asynq.NewTask(ptasks.TaskDMIngest, b)
}

func newDMIngestHandler(fakeDB *fakeDMIngestDTBX) *DMIngestHandler {
	db := dbgen.New(fakeDB)
	r := &runner.Runner{
		DB:     db,
		Loader: wfload.NewLoader(db),
	}
	return NewDMIngestHandler(r, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

const validAccountID = "01010101-0000-0000-0000-000000000001"

func basePayload() ptasks.DMIngestPayload {
	return ptasks.DMIngestPayload{
		AccountID:    validAccountID,
		Source:       "dm",
		Subtype:      "dm",
		MessageID:    "mid-1",
		FromID:       "igsid-user-1",
		FromUsername: "budi",
		Text:         "halo kak",
		EventAt:      "2026-07-08T10:00:00Z",
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestDMIngest_UnknownAccount_SkipsSafely(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{accountErr: errors.New("no rows in result set")}
	h := newDMIngestHandler(fakeDB)

	err := h.ProcessTask(context.Background(), dmIngestTask(t, basePayload()))
	if err != nil {
		t.Fatalf("expected nil error for unknown account, got %v", err)
	}
	if fakeDB.upsertCalls != 0 {
		t.Errorf("expected no conversation upsert for unknown account, got %d calls", fakeDB.upsertCalls)
	}
}

func TestDMIngest_RealAccountDBError_NotSwallowed(t *testing.T) {
	dbErr := errors.New("connection refused")
	fakeDB := &fakeDMIngestDTBX{accountErr: dbErr}
	h := newDMIngestHandler(fakeDB)

	err := h.ProcessTask(context.Background(), dmIngestTask(t, basePayload()))
	if err == nil {
		t.Fatal("expected error on real DB failure, got nil")
	}
}

func TestDMIngest_AccountNotConnected_Skips(t *testing.T) {
	acc := connectedDMAccount()
	acc.Status = "expired"
	fakeDB := &fakeDMIngestDTBX{account: acc}
	h := newDMIngestHandler(fakeDB)

	if err := h.ProcessTask(context.Background(), dmIngestTask(t, basePayload())); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeDB.upsertCalls != 0 {
		t.Errorf("expected no conversation upsert for a non-connected account, got %d calls", fakeDB.upsertCalls)
	}
}

// TestDMIngest_UpsertsConversationOnEveryEvent is the R3 regression guard:
// EVERY messaging event (here, an ordinary "dm" subtype) must upsert the
// conversation window store with the exact contact + timestamp + last_source
// from the payload — with no special-casing for any subtype.
func TestDMIngest_UpsertsConversationOnEveryEvent(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{account: connectedDMAccount()}
	h := newDMIngestHandler(fakeDB)

	p := basePayload()
	if err := h.ProcessTask(context.Background(), dmIngestTask(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if fakeDB.upsertCalls != 1 {
		t.Fatalf("expected exactly 1 conversation upsert, got %d", fakeDB.upsertCalls)
	}

	contactID, _ := fakeDB.lastArgs[1].(string)
	if contactID != p.FromID {
		t.Errorf("upsert contact_ig_user_id = %q, want %q", contactID, p.FromID)
	}
	lastInteractionAt, _ := fakeDB.lastArgs[2].(pgtype.Timestamptz)
	wantTime, _ := time.Parse(time.RFC3339, p.EventAt)
	if !lastInteractionAt.Time.Equal(wantTime) {
		t.Errorf("upsert last_interaction_at = %v, want %v", lastInteractionAt.Time, wantTime)
	}
	lastSource, _ := fakeDB.lastArgs[3].(string)
	if lastSource != "dm" {
		t.Errorf("upsert last_source = %q, want %q", lastSource, "dm")
	}
}

// TestDMIngest_StoryMentionAlsoUpsertsWindow proves R3: story-mention opens
// the window exactly like every other messaging subtype — no branch skips it.
func TestDMIngest_StoryMentionAlsoUpsertsWindow(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{account: connectedDMAccount()}
	h := newDMIngestHandler(fakeDB)

	p := basePayload()
	p.Subtype = "story-mention"
	if err := h.ProcessTask(context.Background(), dmIngestTask(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if fakeDB.upsertCalls != 1 {
		t.Errorf("expected conversation upsert for story-mention (R3), got %d calls", fakeDB.upsertCalls)
	}
}

func TestDMIngest_EventAtFallsBackToNowWhenEmpty(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{account: connectedDMAccount()}
	h := newDMIngestHandler(fakeDB)

	p := basePayload()
	p.EventAt = ""
	before := time.Now().Add(-time.Second)

	if err := h.ProcessTask(context.Background(), dmIngestTask(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	after := time.Now().Add(time.Second)

	lastInteractionAt, _ := fakeDB.lastArgs[2].(pgtype.Timestamptz)
	if lastInteractionAt.Time.Before(before) || lastInteractionAt.Time.After(after) {
		t.Errorf("expected last_interaction_at to fall back to ~now, got %v (window %v..%v)", lastInteractionAt.Time, before, after)
	}
}

func TestDMIngest_UpsertDBError_Propagates(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{account: connectedDMAccount(), upsertErr: errors.New("postgres: connection reset")}
	h := newDMIngestHandler(fakeDB)

	err := h.ProcessTask(context.Background(), dmIngestTask(t, basePayload()))
	if err == nil {
		t.Fatal("expected error when the conversation upsert fails")
	}
}

// TestDMIngest_NoLiveWorkflow_SkipsWithoutError verifies the "no live
// workflow" branch returns cleanly (the fake's emptyRows Query always yields
// zero rows), mirroring comment_ingest.go's analogous branch but WITHOUT the
// legacy fallback (ADR-006 §4.1 — no built-in DM workflow exists).
func TestDMIngest_NoLiveWorkflow_SkipsWithoutError(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{account: connectedDMAccount()}
	h := newDMIngestHandler(fakeDB)

	if err := h.ProcessTask(context.Background(), dmIngestTask(t, basePayload())); err != nil {
		t.Fatalf("expected nil error (no live workflow), got %v", err)
	}
	// Not panicking on Exec proves InsertRun (RunStore.Insert) was never
	// reached — consistent with ADR-004 R2 (only Triggered runs are logged).
}

func TestDMIngest_UnmarshalError(t *testing.T) {
	fakeDB := &fakeDMIngestDTBX{}
	h := newDMIngestHandler(fakeDB)

	task := asynq.NewTask(ptasks.TaskDMIngest, []byte("not-json"))
	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("expected error for malformed payload")
	}
}
