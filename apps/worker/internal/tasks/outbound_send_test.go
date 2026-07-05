package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
}

func (f *fakeOutboundStore) GetAccountByID(_ context.Context, _ pgtype.UUID) (dbgen.Account, error) {
	return f.account, f.accountErr
}
func (f *fakeOutboundStore) GetReservation(_ context.Context, _ pgtype.UUID) (dbgen.Reservation, error) {
	return f.res, f.resErr
}

type fakeGate struct {
	decision workflow.Decision
	err      error
	calls    int
}

func (f *fakeGate) Allow(_ context.Context, _ workflow.OutboundReq) (workflow.Decision, error) {
	f.calls++
	return f.decision, f.err
}

type fakeMarker struct{ called int }

func (f *fakeMarker) MarkWaitingPay(_ context.Context, _ string) error { f.called++; return nil }

type fakeSender struct{ sent int }

func (f *fakeSender) SendPrivateReply(_ context.Context, _, _, _ string) error { f.sent++; return nil }

func connectedAccount() dbgen.Account {
	return dbgen.Account{ID: pgtype.UUID{Bytes: [16]byte{0x01}, Valid: true}, Status: "connected", AccessToken: "tok"}
}

func reservedRes() dbgen.Reservation {
	return dbgen.Reservation{ID: pgtype.UUID{Bytes: [16]byte{0x04}, Valid: true}, Status: dbgen.ReservationStatusReserved}
}

func outboundTask(t *testing.T) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(ptasks.OutboundSendPayload{
		AccountID:     "aaaaaaaa-0000-0000-0000-000000000001",
		IgUserID:      "17841400",
		CommentID:     "comment-1",
		TargetUserID:  "user-1",
		ReservationID: "dddddddd-0000-0000-0000-000000000004",
		ReplyText:     "Halo kak!",
		PostID:        "media-1",
		TriggerKey:    "comment-1",
		CommentAt:     "2026-07-05T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return asynq.NewTask(ptasks.TaskOutboundSend, payload)
}

func newHandler(store *fakeOutboundStore, gate *fakeGate, marker *fakeMarker, sender *fakeSender) *OutboundSendHandler {
	return NewOutboundSendHandler(store, gate, marker,
		func(_ string) PrivateReplySender { return sender }, silentLogger())
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestOutbound_AllowSendsAndMarks(t *testing.T) {
	store := &fakeOutboundStore{account: connectedAccount(), res: reservedRes()}
	gate := &fakeGate{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	marker := &fakeMarker{}
	sender := &fakeSender{}
	h := newHandler(store, gate, marker, sender)

	if err := h.ProcessTask(context.Background(), outboundTask(t)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if sender.sent != 1 {
		t.Errorf("expected 1 send, got %d", sender.sent)
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

	if err := h.ProcessTask(context.Background(), outboundTask(t)); err == nil {
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

	if err := h.ProcessTask(context.Background(), outboundTask(t)); err != nil {
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

	if err := h.ProcessTask(context.Background(), outboundTask(t)); err != nil {
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

	if err := h.ProcessTask(context.Background(), outboundTask(t)); err != nil {
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

	if err := h.ProcessTask(context.Background(), outboundTask(t)); err != nil {
		t.Fatalf("unknown account should drop (nil), got: %v", err)
	}
	if sender.sent != 0 {
		t.Errorf("must not send for unknown account, got %d", sender.sent)
	}
}
