package safety_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zosmed/zosmed/libs/safety"
)

// newTestGate spins up an in-process miniredis and returns a Gate backed by it.
// The miniredis server is automatically closed when the test ends.
func newTestGate(t *testing.T) (safety.Gate, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return safety.New(rdb), mr
}

// newTestGateWithRDB returns the redis.UniversalClient too, needed for kill-switch helpers.
func newTestGateWithRDB(t *testing.T) (safety.Gate, redis.UniversalClient, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return safety.New(rdb), rdb, mr
}

// ── Test: private-reply window expiry ────────────────────────────────────────

// TestPrivateReplyWindowExpiry verifies that a private reply attempted more
// than 7 days after the comment is Rejected (§4c, §10).
func TestPrivateReplyWindowExpiry(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	req := safety.OutboundReq{
		AccountID:    "acc-window-test",
		Kind:         safety.KindPrivateReply,
		TargetUserID: "user1",
		TriggerKey:   "comment-old-1",
		CommentID:    "cmt-old-1",
		CommentAt:    time.Now().Add(-(safety.PrivateReplyWindowDays + 1) * 24 * time.Hour),
	}

	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Reject {
		t.Errorf("expected Reject for expired private-reply window, got %s (reason: %s)", d.Action, d.Reason)
	}
}

// TestPrivateReplyWindowActive verifies that a private reply within the 7-day
// window is Allowed.
func TestPrivateReplyWindowActive(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	req := safety.OutboundReq{
		AccountID:    "acc-window-active",
		Kind:         safety.KindPrivateReply,
		TargetUserID: "user2",
		TriggerKey:   "comment-fresh-1",
		CommentID:    "cmt-fresh-1",
		CommentAt:    time.Now().Add(-2 * 24 * time.Hour), // 2 days ago — within 7-day window
	}

	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Allow {
		t.Errorf("expected Allow within private-reply window, got %s (reason: %s)", d.Action, d.Reason)
	}
}

// ── Test: DM 24h window ───────────────────────────────────────────────────────

// TestDMWindowExpiry verifies that a DM attempted more than 24h after the
// last interaction is Rejected (§4c).
func TestDMWindowExpiry(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	req := safety.OutboundReq{
		AccountID:    "acc-dmwin-test",
		Kind:         safety.KindDM,
		TargetUserID: "user-old",
		TriggerKey:   "dm-trigger-old",
		CommentAt:    time.Now().Add(-(safety.MessagingWindowHours + 1) * time.Hour),
	}

	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Reject {
		t.Errorf("expected Reject for expired DM window, got %s (reason: %s)", d.Action, d.Reason)
	}
}

// ── Test: DM overflow → Queue ─────────────────────────────────────────────────

// TestDMOverflowToQueue verifies that DMs exceeding 200/hr are Queued, never
// Rejected (CLAUDE.md §4c / §10: "overflow → antre, BUKAN ditolak").
func TestDMOverflowToQueue(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	baseTime := time.Now()

	// Send 200 DMs from distinct users so dedupe does not block them.
	// We expect Queue starting at AutoPauseThreshold (80% = 160), then again
	// at the hard cap (200). Either way, none must be Rejected.
	for i := 0; i < 200; i++ {
		req := safety.OutboundReq{
			AccountID:    "acc-dmoverflow",
			Kind:         safety.KindDM,
			TargetUserID: fmt.Sprintf("user-%d", i),
			TriggerKey:   fmt.Sprintf("trigger-%d", i),
			CommentAt:    baseTime,
		}
		d, err := gate.Allow(ctx, req)
		if err != nil {
			t.Fatalf("Allow[%d] error: %v", i, err)
		}
		if d.Action == safety.Reject {
			t.Errorf("DM[%d]: must never Reject on overflow, got Reject (reason: %s)", i, d.Reason)
		}
	}

	// The 201st DM must be Queued (cap exceeded).
	overflow := safety.OutboundReq{
		AccountID:    "acc-dmoverflow",
		Kind:         safety.KindDM,
		TargetUserID: "user-overflow",
		TriggerKey:   "trigger-overflow",
		CommentAt:    baseTime,
	}
	d, err := gate.Allow(ctx, overflow)
	if err != nil {
		t.Fatalf("overflow Allow error: %v", err)
	}
	if d.Action != safety.Queue {
		t.Errorf("expected Queue for DM overflow, got %s (reason: %s)", d.Action, d.Reason)
	}
}

// TestDMAutoPauseAtThreshold verifies that the 161st DM (80% of 200) is Queued
// with an auto-pause reason, not blocked with Reject.
func TestDMAutoPauseAtThreshold(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	// threshold = floor(200 * 0.8) = 160; the 161st must queue.
	threshold := int(float64(160) * safety.AutoPauseThreshold)
	_ = threshold // suppress unused warning; math is: 160 DMs → counter=160 >= 160 → Queue

	baseTime := time.Now()

	// Send exactly AutoPauseThreshold * cap = 160 DMs (all should Allow).
	autoPauseStart := int(float64(200) * safety.AutoPauseThreshold) // = 160
	for i := 0; i < autoPauseStart; i++ {
		req := safety.OutboundReq{
			AccountID:    "acc-autopause",
			Kind:         safety.KindDM,
			TargetUserID: fmt.Sprintf("u%d", i),
			TriggerKey:   fmt.Sprintf("t%d", i),
			CommentAt:    baseTime,
		}
		d, err := gate.Allow(ctx, req)
		if err != nil {
			t.Fatalf("Allow[%d] error: %v", i, err)
		}
		if d.Action != safety.Allow {
			t.Fatalf("DM[%d] expected Allow below threshold, got %s: %s", i, d.Action, d.Reason)
		}
	}

	// The next one (161st) must trigger auto-pause → Queue.
	pauseReq := safety.OutboundReq{
		AccountID:    "acc-autopause",
		Kind:         safety.KindDM,
		TargetUserID: "u-pause",
		TriggerKey:   "t-pause",
		CommentAt:    baseTime,
	}
	d, err := gate.Allow(ctx, pauseReq)
	if err != nil {
		t.Fatalf("auto-pause Allow error: %v", err)
	}
	if d.Action != safety.Queue {
		t.Errorf("expected Queue at auto-pause threshold, got %s (reason: %s)", d.Action, d.Reason)
	}
}

// ── Test: dedupe → Reject ─────────────────────────────────────────────────────

// TestDedupeReject verifies that sending the same (account, user, trigger) twice
// results in Reject on the second attempt (§4c / §10).
func TestDedupeReject(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	req := safety.OutboundReq{
		AccountID:    "acc-dedupe",
		Kind:         safety.KindDM,
		TargetUserID: "user-dup",
		TriggerKey:   "same-trigger",
		CommentAt:    time.Now(),
	}

	// First call: should Allow.
	d1, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("first Allow error: %v", err)
	}
	if d1.Action != safety.Allow {
		t.Fatalf("expected Allow on first send, got %s: %s", d1.Action, d1.Reason)
	}

	// Second call with identical (account, user, trigger): must Reject.
	d2, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("second Allow error: %v", err)
	}
	if d2.Action != safety.Reject {
		t.Errorf("expected Reject on duplicate, got %s (reason: %s)", d2.Action, d2.Reason)
	}
}

// TestDedupeDistinctTriggerAllowed verifies that different TriggerKeys for the
// same user are NOT blocked by dedupe (each trigger is independent).
func TestDedupeDistinctTriggerAllowed(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		req := safety.OutboundReq{
			AccountID:    "acc-dedupe2",
			Kind:         safety.KindDM,
			TargetUserID: "same-user",
			TriggerKey:   fmt.Sprintf("unique-trigger-%d", i),
			CommentAt:    time.Now(),
		}
		d, err := gate.Allow(ctx, req)
		if err != nil {
			t.Fatalf("Allow[%d] error: %v", i, err)
		}
		if d.Action != safety.Allow {
			t.Errorf("distinct trigger[%d] expected Allow, got %s: %s", i, d.Action, d.Reason)
		}
	}
}

// ── Test: kill switch → Reject ────────────────────────────────────────────────

// TestKillSwitchReject verifies that engaging the kill switch causes all
// subsequent Allow calls for that account to Reject (CLAUDE.md §10).
func TestKillSwitchReject(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	// Engage the kill switch.
	if err := safety.EngageKillSwitch(ctx, rdb, "acc-ks"); err != nil {
		t.Fatalf("EngageKillSwitch: %v", err)
	}

	req := safety.OutboundReq{
		AccountID:    "acc-ks",
		Kind:         safety.KindDM,
		TargetUserID: "any-user",
		TriggerKey:   "any-trigger",
		CommentAt:    time.Now(),
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Reject {
		t.Errorf("expected Reject with kill-switch, got %s (reason: %s)", d.Action, d.Reason)
	}
}

// TestKillSwitchDisengage verifies that disengaging the kill switch resumes
// normal Allow behaviour.
func TestKillSwitchDisengage(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	if err := safety.EngageKillSwitch(ctx, rdb, "acc-ks2"); err != nil {
		t.Fatalf("EngageKillSwitch: %v", err)
	}
	if err := safety.DisengageKillSwitch(ctx, rdb, "acc-ks2"); err != nil {
		t.Fatalf("DisengageKillSwitch: %v", err)
	}

	req := safety.OutboundReq{
		AccountID:    "acc-ks2",
		Kind:         safety.KindDM,
		TargetUserID: "user1",
		TriggerKey:   "trigger1",
		CommentAt:    time.Now(),
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Allow {
		t.Errorf("expected Allow after disengage, got %s: %s", d.Action, d.Reason)
	}
}

// ── Burst test: 500 comment replies in 1 minute ───────────────────────────────

// TestBurst500CommentReplies simulates 500 private-reply requests against the
// same post within a single 5-minute window. Expected outcome per §4c / §10:
//   - First 30 → Allow  (commentsPerPostPer5min cap)
//   - Remaining 470 → Queue  (per-post cap exceeded; overflow is queued not rejected)
//   - Zero Reject  (no window/dedupe/kill-switch reason to reject)
func TestBurst500CommentReplies(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	const total = 500
	postID := "post-burst-001"
	baseTime := time.Now()

	var allowed, queued, rejected int

	for i := range total {
		req := safety.OutboundReq{
			AccountID:    "acc-burst",
			Kind:         safety.KindCommentReply,
			TargetUserID: fmt.Sprintf("user-%d", i), // distinct users → no dedupe hit
			TriggerKey:   fmt.Sprintf("cmt-%d", i),  // distinct triggers → no dedupe hit
			CommentID:    fmt.Sprintf("cmt-%d", i),
			CommentAt:    baseTime,
			PostID:       postID,
		}
		d, err := gate.Allow(ctx, req)
		if err != nil {
			t.Fatalf("Allow[%d] error: %v", i, err)
		}
		switch d.Action {
		case safety.Allow:
			allowed++
		case safety.Queue:
			queued++
		case safety.Reject:
			rejected++
			t.Errorf("burst[%d] unexpected Reject: %s", i, d.Reason)
		}
	}

	// commentsPerPostPer5min = 30 → exactly 30 allowed before per-post cap kicks in.
	if allowed != 30 {
		t.Errorf("burst: expected 30 allowed (per-post/5min cap=30), got %d", allowed)
	}
	if queued != total-30 {
		t.Errorf("burst: expected %d queued, got %d", total-30, queued)
	}
	if rejected != 0 {
		t.Errorf("burst: expected 0 rejected, got %d", rejected)
	}

	t.Logf("burst result: allowed=%d queued=%d rejected=%d (total=%d)", allowed, queued, rejected, total)
}

// TestBurst500DMsQueueOverflow simulates 500 DM requests, verifying that
// the first 160 Allow (below 80% auto-pause), 161-200 Queue (auto-pause), and
// 201-500 Queue (cap exceeded) — never Reject.
func TestBurst500DMsQueueOverflow(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	const total = 500
	baseTime := time.Now()

	var allowed, queued, rejected int

	for i := 0; i < total; i++ {
		req := safety.OutboundReq{
			AccountID:    "acc-dmburstall",
			Kind:         safety.KindDM,
			TargetUserID: fmt.Sprintf("u-%d", i),
			TriggerKey:   fmt.Sprintf("t-%d", i),
			CommentAt:    baseTime,
		}
		d, err := gate.Allow(ctx, req)
		if err != nil {
			t.Fatalf("Allow[%d] error: %v", i, err)
		}
		switch d.Action {
		case safety.Allow:
			allowed++
		case safety.Queue:
			queued++
		case safety.Reject:
			rejected++
			t.Errorf("DM burst[%d] unexpected Reject: %s", i, d.Reason)
		}
	}

	// auto-pause threshold = 80% of 200 = 160 → first 160 Allow.
	autoPauseAt := int(float64(200) * safety.AutoPauseThreshold) // 160
	if allowed != autoPauseAt {
		t.Errorf("DM burst: expected %d allowed (below auto-pause), got %d", autoPauseAt, allowed)
	}
	if queued != total-autoPauseAt {
		t.Errorf("DM burst: expected %d queued, got %d", total-autoPauseAt, queued)
	}
	if rejected != 0 {
		t.Errorf("DM burst: expected 0 rejected, got %d", rejected)
	}

	t.Logf("DM burst result: allowed=%d queued=%d rejected=%d (total=%d)", allowed, queued, rejected, total)
}

// ── Test: CurrentUsage ────────────────────────────────────────────────────────

// TestCurrentUsage verifies that gauge counters reflect sent messages.
func TestCurrentUsage(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	// Send 5 DMs.
	for i := 0; i < 5; i++ {
		req := safety.OutboundReq{
			AccountID:    "acc-gauge",
			Kind:         safety.KindDM,
			TargetUserID: fmt.Sprintf("u%d", i),
			TriggerKey:   fmt.Sprintf("t%d", i),
			CommentAt:    time.Now(),
		}
		if _, err := gate.Allow(ctx, req); err != nil {
			t.Fatalf("Allow[%d]: %v", i, err)
		}
	}

	gauges, err := gate.CurrentUsage(ctx, "acc-gauge")
	if err != nil {
		t.Fatalf("CurrentUsage: %v", err)
	}

	var dmHrGauge *safety.QuotaGauge
	for i := range gauges {
		if gauges[i].Key == "dm-hr" {
			dmHrGauge = &gauges[i]
		}
	}
	if dmHrGauge == nil {
		t.Fatal("CurrentUsage: missing dm-hr gauge")
	}
	if dmHrGauge.Used != 5 {
		t.Errorf("dm-hr gauge: expected Used=5, got %d", dmHrGauge.Used)
	}
	if dmHrGauge.Cap != 200 {
		t.Errorf("dm-hr gauge: expected Cap=200, got %d", dmHrGauge.Cap)
	}
}

// ── Test: ordering guarantee ──────────────────────────────────────────────────

// TestCounterNotIncrementedOnReject verifies the ordering guarantee:
// counters must NOT be incremented when a request is Rejected (dedupe, window).
func TestCounterNotIncrementedOnReject(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	req := safety.OutboundReq{
		AccountID:    "acc-order",
		Kind:         safety.KindDM,
		TargetUserID: "u1",
		TriggerKey:   "t1",
		CommentAt:    time.Now(),
	}

	// First call: Allow.
	if _, err := gate.Allow(ctx, req); err != nil {
		t.Fatal(err)
	}

	// Second call: Reject (dedupe). Counter should still be 1, not 2.
	if _, err := gate.Allow(ctx, req); err != nil {
		t.Fatal(err)
	}

	gauges, err := gate.CurrentUsage(ctx, "acc-order")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range gauges {
		if g.Key == "dm-hr" && g.Used != 1 {
			t.Errorf("counter should be 1 after one Allow + one Reject(dedupe), got %d", g.Used)
		}
	}
}
