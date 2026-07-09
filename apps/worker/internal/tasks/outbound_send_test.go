package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/workflow"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeOutboundStore struct {
	account    dbgen.Account
	accountErr error
	res        dbgen.Reservation
	resErr     error

	getReservationCalled bool
}

func (f *fakeOutboundStore) GetAccountByID(_ context.Context, _ pgtype.UUID) (dbgen.Account, error) {
	return f.account, f.accountErr
}
func (f *fakeOutboundStore) GetReservation(_ context.Context, _ pgtype.UUID) (dbgen.Reservation, error) {
	f.getReservationCalled = true
	return f.res, f.resErr
}

type fakeGate struct {
	decision workflow.Decision
	err      error
	calls    int
	lastReq  workflow.OutboundReq
}

func (f *fakeGate) Allow(_ context.Context, req workflow.OutboundReq) (workflow.Decision, error) {
	f.calls++
	f.lastReq = req
	return f.decision, f.err
}

type fakeMarker struct{ called int }

func (f *fakeMarker) MarkWaitingPay(_ context.Context, _ string) error { f.called++; return nil }

// fakeSender implements OutboundSender (ADR-007 §3.6: 3 methods), recording
// which one fired so tests can assert Kind-aware dispatch.
type fakeSender struct {
	sent           int // legacy alias, kept for the pre-existing private-reply tests
	privateReplies int
	commentReplies int
	dms            int

	lastIgUserID string
	lastObjectID string
	lastTargetID string
	lastText     string
}

func (f *fakeSender) SendPrivateReply(_ context.Context, igUserID, commentID, text string) error {
	f.sent++
	f.privateReplies++
	f.lastIgUserID, f.lastObjectID, f.lastText = igUserID, commentID, text
	return nil
}
func (f *fakeSender) ReplyToComment(_ context.Context, commentID, text string) error {
	f.commentReplies++
	f.lastObjectID, f.lastText = commentID, text
	return nil
}
func (f *fakeSender) SendDM(_ context.Context, igUserID, targetUserID, text string) error {
	f.dms++
	f.lastIgUserID, f.lastTargetID, f.lastText = igUserID, targetUserID, text
	return nil
}

func connectedAccount() dbgen.Account {
	return dbgen.Account{ID: pgtype.UUID{Bytes: [16]byte{0x01}, Valid: true}, Status: "connected", AccessToken: "tok"}
}

func reservedRes() dbgen.Reservation {
	return dbgen.Reservation{ID: pgtype.UUID{Bytes: [16]byte{0x04}, Valid: true}, Status: dbgen.ReservationStatusReserved}
}

// outboundBasePayload returns a valid, not-yet-expired private-reply payload
// (ReservationID set, matching the seller kit's usage). Individual tests
// mutate fields via the returned struct before marshalling.
func outboundBasePayload() ptasks.OutboundSendPayload {
	return ptasks.OutboundSendPayload{
		AccountID:     "aaaaaaaa-0000-0000-0000-000000000001",
		Kind:          "private-reply",
		IgUserID:      "17841400",
		ObjectID:      "comment-1",
		TargetUserID:  "user-1",
		ReservationID: "dddddddd-0000-0000-0000-000000000004",
		Text:          "Halo kak!",
		PostID:        "media-1",
		TriggerKey:    "comment-1",
		CommentAt:     "2026-07-05T10:00:00Z",
		Deadline:      time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}
}

func taskFor(t *testing.T, p ptasks.OutboundSendPayload) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return asynq.NewTask(ptasks.TaskOutboundSend, payload)
}

func newHandler(store *fakeOutboundStore, gate *fakeGate, marker *fakeMarker, sender *fakeSender) *OutboundSendHandler {
	return NewOutboundSendHandler(store, gate, marker,
		func(_ string) OutboundSender { return sender }, silentLogger())
}

// ── tests: seller-coupled private-reply (pre-ADR-007 behaviour, now generic) ──

func TestOutbound_AllowSendsAndMarks(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount(), res: reservedRes()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	marker := &fakeMarker{}
	sender := &fakeSender{}
	h := newHandler(store, gate, marker, sender)

	if err := h.ProcessTask(context.Background(), taskFor(t, outboundBasePayload())); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if sender.privateReplies != 1 {
		t.Errorf("expected 1 private reply send, got %d", sender.privateReplies)
	}
	if marker.called != 1 {
		t.Errorf("expected MarkWaitingPay called once, got %d", marker.called)
	}
}

func TestOutbound_QueueAgainReturnsErrorForRetry(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount(), res: reservedRes()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionQueue, Reason: "over quota"}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	if err := h.ProcessTask(context.Background(), taskFor(t, outboundBasePayload())); err == nil {
		t.Fatal("expected error (to trigger asynq retry) when gate still queues")
	}
	if sender.sent != 0 {
		t.Errorf("must not send when gate queues, got %d", sender.sent)
	}
}

func TestOutbound_RejectDrops(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount(), res: reservedRes()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionReject, Reason: "window closed"}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	if err := h.ProcessTask(context.Background(), taskFor(t, outboundBasePayload())); err != nil {
		t.Fatalf("reject should not error (drop), got: %v", err)
	}
	if sender.sent != 0 {
		t.Errorf("must not send on reject, got %d", sender.sent)
	}
}

func TestOutbound_ReservationNotReserved_SkipsWithoutGate(t *testing.T) {
	res := reservedRes()
	res.Status = dbgen.ReservationStatusWaitingPay // already handled
	store := &fakeOutboundStore{account: connectedAccount(), res: res}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	if err := h.ProcessTask(context.Background(), taskFor(t, outboundBasePayload())); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if gate.calls != 0 {
		t.Errorf("gate must not be consulted once reservation left reserved, got %d calls", gate.calls)
	}
	if sender.sent != 0 {
		t.Errorf("must not send a duplicate reply, got %d", sender.sent)
	}
}

func TestOutbound_AccountNotConnected_Drops(t *testing.T) {
	acc := connectedAccount()
	acc.Status = "expired"
	store := &fakeOutboundStore{account: acc, res: reservedRes()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	if err := h.ProcessTask(context.Background(), taskFor(t, outboundBasePayload())); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if gate.calls != 0 || sender.sent != 0 {
		t.Errorf("expired account must skip before gate/send (gate=%d send=%d)", gate.calls, sender.sent)
	}
}

func TestOutbound_UnknownAccount_Drops(t *testing.T) {
	store := &fakeOutboundStore{accountErr: errors.New("no rows in result set")}
	gate := &fakeGate{}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	if err := h.ProcessTask(context.Background(), taskFor(t, outboundBasePayload())); err != nil {
		t.Fatalf("unknown account should drop (nil), got: %v", err)
	}
	if sender.sent != 0 {
		t.Errorf("must not send for unknown account, got %d", sender.sent)
	}
}

// ── tests: ADR-007 #3 generalisation (Kind-aware, Deadline, neutral nodes) ────

// TestOutbound_DeadlinePassed_DropsBeforeAccountOrGate is the §4c TTL
// regression (ADR-007 #3): a stale retry (Deadline already passed) must be
// dropped WITHOUT ever touching the account store or the gate — never sent
// late.
func TestOutbound_DeadlinePassed_DropsBeforeAccountOrGate(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount(), res: reservedRes()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	p := outboundBasePayload()
	p.Deadline = time.Now().Add(-1 * time.Minute).Format(time.RFC3339) // already expired

	if err := h.ProcessTask(context.Background(), taskFor(t, p)); err != nil {
		t.Fatalf("expired deadline should drop (nil), got: %v", err)
	}
	if gate.calls != 0 {
		t.Errorf("gate must NOT be consulted once the §4c deadline has passed, got %d calls", gate.calls)
	}
	if store.getReservationCalled {
		t.Error("reservation store must NOT be consulted once the §4c deadline has passed")
	}
	if sender.sent != 0 {
		t.Errorf("must not send a stale (deadline-passed) retry, got %d", sender.sent)
	}
}

// TestOutbound_KindCommentReply_CallsReplyToComment verifies Kind-aware
// dispatch (ADR-007 §3.6) for a neutral reply-comment retry (no
// ReservationID — the reservation guard must be skipped entirely).
func TestOutbound_KindCommentReply_CallsReplyToComment(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	marker := &fakeMarker{}
	sender := &fakeSender{}
	h := newHandler(store, gate, marker, sender)

	p := outboundBasePayload()
	p.Kind = "comment-reply"
	p.ReservationID = "" // neutral node: never reservation-coupled

	if err := h.ProcessTask(context.Background(), taskFor(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if store.getReservationCalled {
		t.Error("reservation guard must be skipped when ReservationID is empty")
	}
	if sender.commentReplies != 1 {
		t.Errorf("expected ReplyToComment to be called once, got %d", sender.commentReplies)
	}
	if sender.privateReplies != 0 || sender.dms != 0 {
		t.Errorf("only ReplyToComment must fire for Kind=comment-reply, got privateReplies=%d dms=%d",
			sender.privateReplies, sender.dms)
	}
	if marker.called != 0 {
		t.Error("MarkWaitingPay must NOT be called when ReservationID is empty")
	}
	if gate.lastReq.Kind != "comment-reply" {
		t.Errorf("gate req.Kind = %q, want comment-reply", gate.lastReq.Kind)
	}
}

// TestOutbound_KindDM_CallsSendDM verifies Kind-aware dispatch for a neutral
// send-dm retry.
func TestOutbound_KindDM_CallsSendDM(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	p := outboundBasePayload()
	p.Kind = "dm"
	p.ReservationID = ""

	if err := h.ProcessTask(context.Background(), taskFor(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if sender.dms != 1 {
		t.Errorf("expected SendDM to be called once, got %d", sender.dms)
	}
	if sender.privateReplies != 0 || sender.commentReplies != 0 {
		t.Errorf("only SendDM must fire for Kind=dm, got privateReplies=%d commentReplies=%d",
			sender.privateReplies, sender.commentReplies)
	}
}

// TestOutbound_NoReservationID_PrivateReplySkipsReservationGuard verifies the
// neutral send-whatsapp-link path (Kind=private-reply, no ReservationID —
// unlike the seller kit) sends without ever consulting the reservation store
// or the waiting-pay marker.
func TestOutbound_NoReservationID_PrivateReplySkipsReservationGuard(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	marker := &fakeMarker{}
	sender := &fakeSender{}
	h := newHandler(store, gate, marker, sender)

	p := outboundBasePayload()
	p.ReservationID = ""

	if err := h.ProcessTask(context.Background(), taskFor(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if store.getReservationCalled {
		t.Error("reservation guard must be skipped when ReservationID is empty")
	}
	if sender.privateReplies != 1 {
		t.Errorf("expected 1 private reply send, got %d", sender.privateReplies)
	}
	if marker.called != 0 {
		t.Error("MarkWaitingPay must NOT be called when ReservationID is empty")
	}
}

// TestOutbound_UnknownKind_ErrorsRatherThanSendingOrInventingACapability is a
// CLAUDE.md §4b regression guard: sendByKind dispatches on p.Kind through an
// explicit allow-list (comment-reply/private-reply/dm — the only outbound
// shapes §4a permits). An unrecognised Kind must be a retryable error, NEVER
// a silent no-op AND never routed to some other Sender method — this closes
// off the generic retry path (ADR-007 #3) as a place a future node could
// smuggle in a §4b-prohibited "capability" (e.g. a hypothetical
// "follower-notify" or "live-comment" Kind) by simply naming it and relying
// on default dispatch. Every real Kind is produced by this codebase's own
// enqueue closures (libs/workflow/nodes, libs/kits/seller) — never external
// input — so failing loudly here is the correct, safe default.
func TestOutbound_UnknownKind_ErrorsRatherThanSendingOrInventingACapability(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	p := outboundBasePayload()
	p.Kind = "follower-notify" // not in the §4a allow-list — must never be invented
	p.ReservationID = ""

	if err := h.ProcessTask(context.Background(), taskFor(t, p)); err == nil {
		t.Fatal("expected an error for an unrecognised Kind, got nil")
	}
	if sender.sent != 0 || sender.commentReplies != 0 || sender.dms != 0 {
		t.Errorf("an unrecognised Kind must NEVER dispatch to any Sender method, got privateReplies=%d commentReplies=%d dms=%d",
			sender.privateReplies, sender.commentReplies, sender.dms)
	}
}

// TestOutbound_GateConsultedBeforeSend proves the §10 one-door ordering: the
// gate is called, and only on DecisionAllow does the sender fire.
func TestOutbound_GateConsultedBeforeSend(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &fakeSender{}
	h := newHandler(store, gate, &fakeMarker{}, sender)

	p := outboundBasePayload()
	p.Kind = "dm"
	p.ReservationID = ""

	if err := h.ProcessTask(context.Background(), taskFor(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if gate.calls != 1 {
		t.Fatalf("expected exactly 1 gate.Allow call, got %d", gate.calls)
	}
	if sender.dms != 1 {
		t.Fatal("expected SendDM to fire after gate.Allow returned DecisionAllow")
	}
}
