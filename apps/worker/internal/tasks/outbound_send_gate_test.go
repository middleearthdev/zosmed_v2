package tasks

// outbound_send_gate_test.go — ADR-007 §5 scenarios 1, 2, 4 (reinforced), 5,
// 13, 14: OutboundSendHandler wired against the REAL safety.Gate (miniredis-
// backed, same pattern as libs/safety/gate_test.go) instead of the
// outbound_send_test.go fakeGate. Those scenarios specifically exercise Gate
// behaviour (quota buckets, kill-switch, the 7-day private-reply window) that
// a scripted fakeGate cannot prove — only the real libs/safety implementation
// can.
//
// realGateAdapter below duplicates the ~15-line Decision/Action mapping in
// apps/worker/internal/runner's unexported gateAdapter. That type lives in a
// different package (runner, not tasks) so it cannot be reused directly;
// duplicating this small, stable adapter here (rather than exporting internal
// runner plumbing just for a test) is the smaller violation (§12a-4).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/zosmed/zosmed/libs/safety"
	"github.com/zosmed/zosmed/libs/workflow"
)

// ── real-Gate test harness ────────────────────────────────────────────────────

// realGateAdapter adapts safety.Gate to workflow.Gater — mirrors
// apps/worker/internal/runner's gateAdapter (kept private to that package).
type realGateAdapter struct{ g safety.Gate }

func (a *realGateAdapter) Allow(ctx context.Context, req workflow.OutboundReq) (workflow.Decision, error) {
	d, err := a.g.Allow(ctx, safety.OutboundReq{
		AccountID:    req.AccountID,
		Kind:         req.Kind,
		TargetUserID: req.TargetUserID,
		TriggerKey:   req.TriggerKey,
		CommentID:    req.CommentID,
		CommentAt:    req.CommentAt,
		PostID:       req.PostID,
	})
	if err != nil {
		return workflow.Decision{}, err
	}
	var action workflow.DecisionAction
	switch d.Action {
	case safety.Allow:
		action = workflow.DecisionAllow
	case safety.Queue:
		action = workflow.DecisionQueue
	default:
		action = workflow.DecisionReject
	}
	return workflow.Decision{Action: action, Reason: d.Reason}, nil
}

// newRealGate spins up an in-process miniredis and returns both the real
// Gate (workflow.Gater, for the handler) and the raw redis.UniversalClient
// (for kill-switch helpers / cross-checking).
func newRealGate(t *testing.T) (workflow.Gater, redis.UniversalClient) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return &realGateAdapter{g: safety.New(rdb)}, rdb
}

// saturateDMHourly consumes n Allow calls of Kind=dm for accountID with
// distinct users/triggers, modelling "other DM traffic already sent this
// hour" so the NEXT Allow call queues (§4c: 200/hr cap).
func saturateDMHourly(t *testing.T, ctx context.Context, gate workflow.Gater, accountID string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		d, err := gate.Allow(ctx, workflow.OutboundReq{
			AccountID:    accountID,
			Kind:         "dm",
			TargetUserID: fmt.Sprintf("filler-user-%d", i),
			TriggerKey:   fmt.Sprintf("filler-trigger-%d", i),
			CommentAt:    time.Now(),
		})
		if err != nil {
			t.Fatalf("saturateDMHourly[%d]: %v", i, err)
		}
		if d.Action == workflow.DecisionReject {
			t.Fatalf("saturateDMHourly[%d]: unexpected Reject: %s", i, d.Reason)
		}
	}
}

// saturateCommentReplyHourly is the comment-reply mirror of saturateDMHourly
// (§4c: 750/hr cap). PostID is deliberately left empty so only the /hr cap is
// exercised (not the 30/post/5min cap).
func saturateCommentReplyHourly(t *testing.T, ctx context.Context, gate workflow.Gater, accountID string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		d, err := gate.Allow(ctx, workflow.OutboundReq{
			AccountID:    accountID,
			Kind:         "comment-reply",
			TargetUserID: fmt.Sprintf("filler-user-%d", i),
			TriggerKey:   fmt.Sprintf("filler-comment-%d", i),
			CommentID:    fmt.Sprintf("filler-comment-%d", i),
			CommentAt:    time.Now(),
		})
		if err != nil {
			t.Fatalf("saturateCommentReplyHourly[%d]: %v", i, err)
		}
		if d.Action == workflow.DecisionReject {
			t.Fatalf("saturateCommentReplyHourly[%d]: unexpected Reject: %s", i, d.Reason)
		}
	}
}

// ── Scenario 1: send-dm Queue → dequeue after quota recovers → Allow → exactly one SendDM ──

func TestOutboundGate_DMQueue_RecoversAfterQuotaReset_SendsExactlyOnce(t *testing.T) {
	gate, rdb := newRealGate(t)
	ctx := context.Background()
	const accountID = "aaaaaaaa-0000-0000-0000-0000000000d1"

	// Saturate the DM/hr bucket with OTHER traffic so the payload under test
	// queues on its first attempt. The auto-pause threshold (80% of the 200
	// cap = 160, libs/safety/quota.go) already triggers Queue, so 160 fillers
	// suffice — no need to reach the hard 200 cap.
	saturateDMHourly(t, ctx, gate, accountID, int(float64(200)*safety.AutoPauseThreshold))

	store := &fakeOutboundStore{account: connectedAccount()}
	sender := &fakeSender{}
	h := NewOutboundSendHandler(store, gate, &fakeMarker{}, func(_ string) OutboundSender { return sender }, silentLogger())

	p := outboundBasePayload()
	p.AccountID = accountID
	p.Kind = "dm"
	p.ReservationID = ""
	p.CommentAt = time.Now().Format(time.RFC3339) // fresh — well within the 24h DM window
	p.Deadline = time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	// Scenario 4 (reinforced with a REAL gate): dequeue while still over quota
	// must return an error (asynq retries) and must NOT send.
	if err := h.ProcessTask(ctx, taskFor(t, p)); err == nil {
		t.Fatal("expected an error (gate still Queue) to trigger an asynq retry")
	}
	if sender.dms != 0 {
		t.Fatalf("must not send while gate still queues, got %d", sender.dms)
	}

	// "Quota recovers": in production this is the fixed-window bucket rolling
	// over to the next hour; here we flush Redis to model the same effect (a
	// fresh, empty bucket) without needing to fast-forward wall-clock time
	// (the bucket key is derived from time.Now(), which miniredis cannot fake
	// — see libs/safety/quota.go).
	flushRedis(t, rdb)

	// Dequeue (retry) after recovery: must Allow and send exactly once.
	if err := h.ProcessTask(ctx, taskFor(t, p)); err != nil {
		t.Fatalf("expected the retry to succeed after quota recovery, got: %v", err)
	}
	if sender.dms != 1 {
		t.Fatalf("expected SendDM called exactly once after recovery, got %d", sender.dms)
	}
}

// ── Scenario 2: reply-comment Queue (750/hr) → dequeue re-checks the real Gate ──

func TestOutboundGate_CommentReplyQueue_RecoversAfterQuotaReset_SendsExactlyOnce(t *testing.T) {
	gate, rdb := newRealGate(t)
	ctx := context.Background()
	const accountID = "aaaaaaaa-0000-0000-0000-0000000000d2"

	saturateCommentReplyHourly(t, ctx, gate, accountID, 750)

	store := &fakeOutboundStore{account: connectedAccount()}
	sender := &fakeSender{}
	h := NewOutboundSendHandler(store, gate, &fakeMarker{}, func(_ string) OutboundSender { return sender }, silentLogger())

	p := outboundBasePayload()
	p.AccountID = accountID
	p.Kind = "comment-reply"
	p.ReservationID = ""
	p.CommentAt = time.Now().Format(time.RFC3339)
	p.Deadline = time.Now().Add(6 * time.Hour).Format(time.RFC3339) // DeferredCommentReplyTTL-shaped

	if err := h.ProcessTask(ctx, taskFor(t, p)); err == nil {
		t.Fatal("expected an error (gate still Queue at 750/hr cap) to trigger an asynq retry")
	}
	if sender.commentReplies != 0 {
		t.Fatalf("must not send while gate still queues, got %d", sender.commentReplies)
	}

	flushRedis(t, rdb)

	if err := h.ProcessTask(ctx, taskFor(t, p)); err != nil {
		t.Fatalf("expected the retry to succeed after quota recovery, got: %v", err)
	}
	if sender.commentReplies != 1 {
		t.Fatalf("expected ReplyToComment called exactly once after recovery, got %d", sender.commentReplies)
	}
}

// ── Scenario 5: re-Gate.Allow at dequeue is called strictly BEFORE the sender ──

// orderRecorder is shared by a gate wrapper and a sender wrapper to prove
// call ORDER (not just call counts) — §10 one-door.
type orderRecorder struct{ calls []string }

type orderedGate struct {
	inner workflow.Gater
	rec   *orderRecorder
}

func (g *orderedGate) Allow(ctx context.Context, req workflow.OutboundReq) (workflow.Decision, error) {
	g.rec.calls = append(g.rec.calls, "gate")
	return g.inner.Allow(ctx, req)
}

type orderedSender struct {
	inner OutboundSender
	rec   *orderRecorder
}

func (s *orderedSender) ReplyToComment(ctx context.Context, commentID, text string) error {
	s.rec.calls = append(s.rec.calls, "send")
	return s.inner.ReplyToComment(ctx, commentID, text)
}
func (s *orderedSender) SendPrivateReply(ctx context.Context, igUserID, commentID, text string) error {
	s.rec.calls = append(s.rec.calls, "send")
	return s.inner.SendPrivateReply(ctx, igUserID, commentID, text)
}
func (s *orderedSender) SendDM(ctx context.Context, igUserID, targetUserID, text string) error {
	s.rec.calls = append(s.rec.calls, "send")
	return s.inner.SendDM(ctx, igUserID, targetUserID, text)
}

func TestOutboundGate_GateCalledStrictlyBeforeSend_OrderRecorded(t *testing.T) {
	gate, _ := newRealGate(t)
	ctx := context.Background()

	rec := &orderRecorder{}
	store := &fakeOutboundStore{account: connectedAccount()}
	innerSender := &fakeSender{}
	h := NewOutboundSendHandler(store, &orderedGate{inner: gate, rec: rec}, &fakeMarker{},
		func(_ string) OutboundSender { return &orderedSender{inner: innerSender, rec: rec} }, silentLogger())

	p := outboundBasePayload()
	p.AccountID = "aaaaaaaa-0000-0000-0000-0000000000d5"
	p.Kind = "dm"
	p.ReservationID = ""
	p.CommentAt = time.Now().Format(time.RFC3339)
	p.Deadline = time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	if err := h.ProcessTask(ctx, taskFor(t, p)); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if len(rec.calls) != 2 || rec.calls[0] != "gate" || rec.calls[1] != "send" {
		t.Fatalf("expected call order [gate, send], got %v", rec.calls)
	}
}

// ── Scenario 13: kill-switch engaged between enqueue and dequeue → Reject → drop ──

func TestOutboundGate_KillSwitchEngagedBeforeDequeue_RejectsAndDrops(t *testing.T) {
	gate, rdb := newRealGate(t)
	ctx := context.Background()
	const accountID = "aaaaaaaa-0000-0000-0000-0000000000d3"

	// "Enqueue" happened earlier (represented simply by the payload existing);
	// the kill switch is engaged AFTER that but BEFORE this dequeue attempt.
	if err := safety.EngageKillSwitch(ctx, rdb, accountID); err != nil {
		t.Fatalf("EngageKillSwitch: %v", err)
	}

	store := &fakeOutboundStore{account: connectedAccount()}
	sender := &fakeSender{}
	h := NewOutboundSendHandler(store, gate, &fakeMarker{}, func(_ string) OutboundSender { return sender }, silentLogger())

	p := outboundBasePayload()
	p.AccountID = accountID
	p.Kind = "private-reply"
	p.ReservationID = ""
	p.CommentAt = time.Now().Format(time.RFC3339)
	p.Deadline = time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	if err := h.ProcessTask(ctx, taskFor(t, p)); err != nil {
		t.Fatalf("kill-switch Reject must drop (nil), not retry, got: %v", err)
	}
	if sender.sent != 0 {
		t.Fatalf("must not send once the kill-switch is engaged, got %d", sender.sent)
	}
}

// ── Scenario 14: CommentAt >7 days at dequeue → Gate window Reject → drop (never sent late) ──

// This is DISTINCT from the existing fakeGate TestOutbound_DeadlinePassed_
// DropsBeforeAccountOrGate (scenario 3, the handler's own §4c Deadline
// pre-check): here the payload's Deadline is still in the FUTURE (so the
// handler's own TTL guard does NOT fire), yet the real Gate's window check
// must independently Reject because CommentAt is more than 7 days old.
func TestOutboundGate_PrivateReplyCommentAtOver7Days_GateRejectsWindowNotDeadline(t *testing.T) {
	gate, _ := newRealGate(t)
	ctx := context.Background()

	store := &fakeOutboundStore{account: connectedAccount()}
	sender := &fakeSender{}
	h := NewOutboundSendHandler(store, gate, &fakeMarker{}, func(_ string) OutboundSender { return sender }, silentLogger())

	p := outboundBasePayload()
	p.AccountID = "aaaaaaaa-0000-0000-0000-0000000000d4"
	p.Kind = "private-reply"
	p.ReservationID = ""
	p.CommentAt = time.Now().Add(-8 * 24 * time.Hour).Format(time.RFC3339) // §4c: past the 7-day window
	p.Deadline = time.Now().Add(1 * time.Hour).Format(time.RFC3339)        // handler's own TTL NOT expired

	if err := h.ProcessTask(ctx, taskFor(t, p)); err != nil {
		t.Fatalf("window-expired Reject must drop (nil), not retry, got: %v", err)
	}
	if sender.privateReplies != 0 {
		t.Fatalf("must NOT send a private reply past the 7-day window, got %d", sender.privateReplies)
	}
}

// ── small local helpers ───────────────────────────────────────────────────────

// flushRedis resets every counter/dedupe/kill-switch key, modelling a
// fixed-window quota bucket rollover (see the comment on
// TestOutboundGate_DMQueue_RecoversAfterQuotaReset_SendsExactlyOnce). Plain
// redis protocol — works identically against miniredis or a real server.
func flushRedis(t *testing.T, rdb redis.UniversalClient) {
	t.Helper()
	if err := rdb.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flushRedis: %v", err)
	}
}
