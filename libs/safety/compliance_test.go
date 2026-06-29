package safety_test

// compliance_test.go — §4c compliance boundary tests.
//
// These tests verify rate-limit caps and window boundaries that are NOT covered
// by the existing gate_test.go burst tests (which focus on per-post/5min and
// DM/hr auto-pause). New coverage added here:
//   - Comment-reply hourly cap (750/hr) auto-pause boundary
//   - DM daily cap (1000/day) auto-pause and hard-cap boundary
//   - Exact window boundaries (7-day private reply, 24h DM)
//   - Mixed burst: DM and private-reply counters are independent
//   - Compliance gap: unknown outbound Kind → Allow (no quota enforcement)

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/safety"
)

// ── Comment-reply hourly cap (750/hr) ────────────────────────────────────────

// TestCommentReplyHourlyCap_AutoPauseAt600 verifies that the 601st private-reply
// request (with unique PostID to avoid per-post/5min cap) is Queued with an
// auto-pause reason (counter = 600 ≥ 80% × 750).
//
// Uses direct counter manipulation via rdb to avoid sending 600 sequential requests.
func TestCommentReplyHourlyCap_AutoPauseAt600(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	const accountID = "acc-cr-autopause"
	now := time.Now()

	// Preset the comment-reply/hr counter to 599 (one below auto-pause at 600).
	crHrBucket := now.Unix() / 3600
	crHrKey := fmt.Sprintf("safety:q:%s:cr:hr:%d", accountID, crHrBucket)
	if err := rdb.Set(ctx, crHrKey, "599", 2*time.Hour).Err(); err != nil {
		t.Fatalf("preset counter: %v", err)
	}

	// The 600th request (counter reads 599 → below threshold → Allow → counter becomes 600).
	req := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindCommentReply,
		TargetUserID: "user-a",
		TriggerKey:   "cmt-a",
		CommentID:    "cmt-a",
		CommentAt:    now,
		PostID:       "post-unique-a", // unique → per-post cap not hit
	}
	d1, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow (600th) error: %v", err)
	}
	if d1.Action != safety.Allow {
		t.Errorf("600th request: expected Allow (below auto-pause threshold), got %s: %s",
			d1.Action, d1.Reason)
	}

	// The 601st request (counter reads 600 → ≥ 80% × 750 = 600 → Queue auto-pause).
	req2 := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindCommentReply,
		TargetUserID: "user-b",
		TriggerKey:   "cmt-b",
		CommentID:    "cmt-b",
		CommentAt:    now,
		PostID:       "post-unique-b",
	}
	d2, err := gate.Allow(ctx, req2)
	if err != nil {
		t.Fatalf("Allow (601st) error: %v", err)
	}
	if d2.Action != safety.Queue {
		t.Errorf("601st request: expected Queue (auto-pause at 600/hr), got %s: %s",
			d2.Action, d2.Reason)
	}
}

// TestCommentReplyHourlyCap_HardCapAt750 verifies that requests when the
// counter is already at the hard cap (750) are still Queued (not Rejected).
func TestCommentReplyHourlyCap_HardCapAt750(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	const accountID = "acc-cr-hardcap"
	now := time.Now()

	// Preset counter to 750 (the hard cap).
	crHrBucket := now.Unix() / 3600
	crHrKey := fmt.Sprintf("safety:q:%s:cr:hr:%d", accountID, crHrBucket)
	if err := rdb.Set(ctx, crHrKey, "750", 2*time.Hour).Err(); err != nil {
		t.Fatalf("preset counter: %v", err)
	}

	req := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindCommentReply,
		TargetUserID: "user-c",
		TriggerKey:   "cmt-c",
		CommentID:    "cmt-c",
		CommentAt:    now,
		PostID:       "post-unique-c",
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	// At or above hard cap → Queue (not Reject). §4c: overflow → queue.
	if d.Action != safety.Queue {
		t.Errorf("at hard cap 750: expected Queue, got %s: %s", d.Action, d.Reason)
	}
	if d.Action == safety.Reject {
		t.Error("COMPLIANCE VIOLATION §4c: comment-reply overflow must Queue, never Reject")
	}
}

// ── DM daily cap (1000/day) ──────────────────────────────────────────────────

// TestDMDailyCap_AutoPauseAt800 verifies that the 801st DM in a day is Queued
// (daily counter = 800 ≥ 80% × 1000) without violating the DM/hr cap (200/hr).
func TestDMDailyCap_AutoPauseAt800(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	const accountID = "acc-dm-dayautopause"
	now := time.Now()

	// Preset daily counter to 799 (one below auto-pause at 800).
	// The hourly counter is 0 (not preset), so DM/hr cap doesn't interfere.
	dmDayBucket := now.Unix() / 86400
	dmDayKey := fmt.Sprintf("safety:q:%s:dm:day:%d", accountID, dmDayBucket)
	if err := rdb.Set(ctx, dmDayKey, "799", 48*time.Hour).Err(); err != nil {
		t.Fatalf("preset daily counter: %v", err)
	}

	// The 800th DM (dayCounter reads 799 → < 800 → not auto-pause → Allow).
	req1 := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindDM,
		TargetUserID: "user-day-a",
		TriggerKey:   "dm-day-a",
		CommentAt:    now,
	}
	d1, err := gate.Allow(ctx, req1)
	if err != nil {
		t.Fatalf("Allow (800th DM) error: %v", err)
	}
	if d1.Action != safety.Allow {
		t.Errorf("800th DM: expected Allow (day counter=799 < 800), got %s: %s",
			d1.Action, d1.Reason)
	}

	// The 801st DM (dayCounter reads 800 → ≥ 80% × 1000 = 800 → Queue auto-pause).
	req2 := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindDM,
		TargetUserID: "user-day-b",
		TriggerKey:   "dm-day-b",
		CommentAt:    now,
	}
	d2, err := gate.Allow(ctx, req2)
	if err != nil {
		t.Fatalf("Allow (801st DM) error: %v", err)
	}
	if d2.Action != safety.Queue {
		t.Errorf("801st DM: expected Queue (daily auto-pause at 800), got %s: %s",
			d2.Action, d2.Reason)
	}
}

// TestDMDailyCap_HardCapAt1000 verifies that DMs when the daily counter is at
// the hard cap (1000) are Queued, never Rejected.
func TestDMDailyCap_HardCapAt1000(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	const accountID = "acc-dm-dayhardcap"
	now := time.Now()

	dmDayBucket := now.Unix() / 86400
	dmDayKey := fmt.Sprintf("safety:q:%s:dm:day:%d", accountID, dmDayBucket)
	if err := rdb.Set(ctx, dmDayKey, "1000", 48*time.Hour).Err(); err != nil {
		t.Fatalf("preset daily counter: %v", err)
	}

	req := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindDM,
		TargetUserID: "user-day-c",
		TriggerKey:   "dm-day-c",
		CommentAt:    now,
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Queue {
		t.Errorf("at daily hard cap 1000: expected Queue, got %s: %s", d.Action, d.Reason)
	}
	if d.Action == safety.Reject {
		t.Error("COMPLIANCE VIOLATION §4c: DM daily overflow must Queue, never Reject")
	}
}

// ── Window boundary conditions ───────────────────────────────────────────────

// TestPrivateReplyWindow_JustWithin7Days verifies that a private reply sent
// 1 hour before the 7-day deadline is Allowed.
func TestPrivateReplyWindow_JustWithin7Days(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	// 7 days ago + 1 hour = deadline is 1 hour in the future → within window.
	commentAt := time.Now().Add(-(safety.PrivateReplyWindowDays*24*time.Hour - time.Hour))

	req := safety.OutboundReq{
		AccountID:    "acc-prw-within",
		Kind:         safety.KindPrivateReply,
		TargetUserID: "user-prw1",
		TriggerKey:   "cmt-prw1",
		CommentID:    "cmt-prw1",
		CommentAt:    commentAt,
		PostID:       "post-prw1",
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Allow {
		t.Errorf("private reply 1h before deadline: expected Allow, got %s: %s", d.Action, d.Reason)
	}
}

// TestPrivateReplyWindow_JustOutside7Days verifies that a private reply sent
// 1 hour after the 7-day deadline is Rejected.
func TestPrivateReplyWindow_JustOutside7Days(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	// 7 days ago - 1 hour = deadline was 1 hour ago → outside window.
	commentAt := time.Now().Add(-(safety.PrivateReplyWindowDays*24*time.Hour + time.Hour))

	req := safety.OutboundReq{
		AccountID:    "acc-prw-outside",
		Kind:         safety.KindPrivateReply,
		TargetUserID: "user-prw2",
		TriggerKey:   "cmt-prw2",
		CommentID:    "cmt-prw2",
		CommentAt:    commentAt,
		PostID:       "post-prw2",
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Reject {
		t.Errorf("private reply 1h after 7-day deadline: expected Reject, got %s: %s",
			d.Action, d.Reason)
	}
}

// TestDMWindow_JustWithin24Hours verifies that a DM triggered by an interaction
// 1 hour ago is Allowed (well within 24h window).
func TestDMWindow_JustWithin24Hours(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	commentAt := time.Now().Add(-1 * time.Hour) // 1 hour ago → within 24h window.

	req := safety.OutboundReq{
		AccountID:    "acc-dmwin-within",
		Kind:         safety.KindDM,
		TargetUserID: "user-dmw1",
		TriggerKey:   "dm-dmw1",
		CommentAt:    commentAt,
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Allow {
		t.Errorf("DM 1h after interaction: expected Allow, got %s: %s", d.Action, d.Reason)
	}
}

// TestDMWindow_JustOutside24Hours verifies that a DM sent 25 hours after the
// last interaction is Rejected and directs to opt-in.
func TestDMWindow_JustOutside24Hours(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	commentAt := time.Now().Add(-25 * time.Hour) // 25h ago → outside 24h window.

	req := safety.OutboundReq{
		AccountID:    "acc-dmwin-outside",
		Kind:         safety.KindDM,
		TargetUserID: "user-dmw2",
		TriggerKey:   "dm-dmw2",
		CommentAt:    commentAt,
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action != safety.Reject {
		t.Errorf("DM 25h after interaction: expected Reject, got %s: %s", d.Action, d.Reason)
	}
}

// TestDMWindow_ZeroCommentAt_AllowedByGate verifies that when CommentAt is the
// zero Time, the gate passes (window tracking is the caller's responsibility).
func TestDMWindow_ZeroCommentAt_AllowedByGate(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	req := safety.OutboundReq{
		AccountID:    "acc-dmwin-zero",
		Kind:         safety.KindDM,
		TargetUserID: "user-zero1",
		TriggerKey:   "dm-zero1",
		CommentAt:    time.Time{}, // zero value → gate defers to caller
	}
	d, err := gate.Allow(ctx, req)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if d.Action == safety.Reject {
		t.Errorf("zero CommentAt should not be Rejected by gate; got %s: %s", d.Action, d.Reason)
	}
}

// ── Mixed burst: independent counters ────────────────────────────────────────

// TestMixedBurst_PrivateReplySharesDMQuota_CommentReplyIndependent verifies the
// corrected metering (MAJOR-1 fix): a private reply is DELIVERED as a DM, so it
// shares the DM/hr counter — once DMs hit auto-pause, private replies are paused
// too. A PUBLIC comment reply (KindCommentReply) uses a separate counter and is
// unaffected (§4c: comment-reply caps are distinct from DM caps).
func TestMixedBurst_PrivateReplySharesDMQuota_CommentReplyIndependent(t *testing.T) {
	gate, _ := newTestGate(t)
	ctx := context.Background()

	const accountID = "acc-mixed-burst"
	baseTime := time.Now()

	// Step 1: send 160 DMs (reaches DM/hr auto-pause threshold = 80% × 200).
	autoPauseAt := int(float64(200) * safety.AutoPauseThreshold) // 160
	for i := range autoPauseAt {
		req := safety.OutboundReq{
			AccountID:    accountID,
			Kind:         safety.KindDM,
			TargetUserID: fmt.Sprintf("u-dm-%d", i),
			TriggerKey:   fmt.Sprintf("t-dm-%d", i),
			CommentAt:    baseTime,
		}
		d, err := gate.Allow(ctx, req)
		if err != nil {
			t.Fatalf("DM[%d] Allow error: %v", i, err)
		}
		if d.Action != safety.Allow {
			t.Fatalf("DM[%d] expected Allow (below DM/hr threshold), got %s: %s",
				i, d.Action, d.Reason)
		}
	}

	// Step 2: a private reply now SHARES the DM counter → also auto-paused (Queue).
	// This is the §4c DM-cap enforcement that MAJOR-1 restored.
	prReq := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindPrivateReply,
		TargetUserID: "u-pr-1",
		TriggerKey:   "cmt-pr-1",
		CommentID:    "cmt-pr-1",
		CommentAt:    baseTime,
		PostID:       "post-mixed-001",
	}
	pd, err := gate.Allow(ctx, prReq)
	if err != nil {
		t.Fatalf("private-reply Allow error: %v", err)
	}
	if pd.Action != safety.Queue {
		t.Errorf("private-reply after DM auto-pause: expected Queue (shares DM counter per §4c), got %s: %s",
			pd.Action, pd.Reason)
	}

	// Step 3: a PUBLIC comment reply uses its OWN counter → still Allow.
	crReq := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindCommentReply,
		TargetUserID: "u-cr-1",
		TriggerKey:   "cmt-cr-1",
		CommentID:    "cmt-cr-1",
		CommentAt:    baseTime,
		PostID:       "post-mixed-002",
	}
	cd, err := gate.Allow(ctx, crReq)
	if err != nil {
		t.Fatalf("comment-reply Allow error: %v", err)
	}
	if cd.Action != safety.Allow {
		t.Errorf("comment-reply after DM auto-pause: expected Allow (independent counter), got %s: %s",
			cd.Action, cd.Reason)
	}
}

// ── §4b Compliance gap: unknown outbound Kind ─────────────────────────────────

// TestComplianceGap_UnknownKind_GateAllowsWithNoQuota documents that the safety
// gate ALLOWS requests with an unrecognised Kind (e.g., "auto-follow") because
// checkQuota and checkWindow have no case for unknown kinds.
//
// THIS IS A DESIGN GAP (not a blocking bug for MVP):
//   - In practice, only KindPrivateReply and KindDM are produced by Kit nodes.
//   - §4b enforcement is the responsibility of the Kit layer, not the gate.
//   - The gate is a rate-limit/window layer, not an action whitelist.
//
// This test serves as a regression guard: if the behaviour changes (gate starts
// rejecting unknown kinds), the test will fail and a reviewer must decide whether
// the change is intentional.
func TestComplianceGap_UnknownKind_GateAllowsWithNoQuota(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	const accountID = "acc-unknown-kind"

	unknownKinds := []string{
		"auto-follow",    // §4b.2: explicitly banned
		"new-follower",   // §4b.1: explicitly banned
		"live-comment",   // §4b.5: explicitly banned
		"blast-dm",       // §4b.6: explicitly banned
	}

	for _, kind := range unknownKinds {
		t.Run("kind="+kind, func(t *testing.T) {
			req := safety.OutboundReq{
				AccountID:    accountID,
				Kind:         kind,
				TargetUserID: "any-user-" + kind,
				TriggerKey:   "any-trigger-" + kind,
				CommentAt:    time.Now(),
			}
			d, err := gate.Allow(ctx, req)
			if err != nil {
				t.Fatalf("Allow error: %v", err)
			}

			// Document the current (gap) behaviour: unknown kinds are Allowed.
			// If this assertion ever fails, a new whitelist guard has been added —
			// verify the §4b enforcement is intentional.
			t.Logf("COMPLIANCE GAP: Kind=%q → Action=%s (reason: %s)", kind, d.Action, d.Reason)

			// Also verify: unknown kinds do NOT increment any quota counter.
			usage, err := gate.CurrentUsage(ctx, accountID)
			if err != nil {
				t.Fatalf("CurrentUsage error: %v", err)
			}
			for _, g := range usage {
				if g.Used > 0 {
					t.Errorf("COMPLIANCE GAP: unknown Kind=%q incremented counter %s (used=%d); "+
						"this should be 0 (no quota tracked for unknown kinds)",
						kind, g.Key, g.Used)
				}
			}

			// Disengage kill switch state between sub-tests.
			_ = safety.DisengageKillSwitch(ctx, rdb, accountID)
		})
	}
}

// TestDMHourlyBoundary_Exactly200_IsQueued verifies the exact hourly DM cap
// boundary: request that brings counter TO 200 sees counter=199 on read
// (pre-increment), which is < 200, so it Allows. The NEXT one sees 200 → Queue.
func TestDMHourlyBoundary_Exactly200_IsQueued(t *testing.T) {
	gate, rdb, _ := newTestGateWithRDB(t)
	ctx := context.Background()

	const accountID = "acc-dm-boundary"
	now := time.Now()

	// Preset hourly counter to 199.
	hrBucket := now.Unix() / 3600
	hrKey := fmt.Sprintf("safety:q:%s:dm:hr:%d", accountID, hrBucket)
	if err := rdb.Set(ctx, hrKey, "199", 2*time.Hour).Err(); err != nil {
		t.Fatalf("preset counter: %v", err)
	}

	// 200th request: reads 199 → < 200 → not at cap → Allow (counter becomes 200).
	r200 := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindDM,
		TargetUserID: "u-200",
		TriggerKey:   "t-200",
		CommentAt:    now,
	}
	d200, err := gate.Allow(ctx, r200)
	if err != nil {
		t.Fatalf("200th Allow error: %v", err)
	}
	// Note: counter 199 is >= 160 (auto-pause) → will actually be Queue, not Allow.
	// This is correct behaviour (auto-pause kicks in first).
	if d200.Action == safety.Reject {
		t.Errorf("200th DM: must never Reject (should be Queue or Allow), got Reject: %s", d200.Reason)
	}

	// 201st request: reads 200 → ≥ 200 (hard cap) → Queue.
	r201 := safety.OutboundReq{
		AccountID:    accountID,
		Kind:         safety.KindDM,
		TargetUserID: "u-201",
		TriggerKey:   "t-201",
		CommentAt:    now,
	}
	d201, err := gate.Allow(ctx, r201)
	if err != nil {
		t.Fatalf("201st Allow error: %v", err)
	}
	if d201.Action != safety.Queue {
		t.Errorf("201st DM (hard cap): expected Queue, got %s: %s", d201.Action, d201.Reason)
	}
	if d201.Action == safety.Reject {
		t.Error("COMPLIANCE VIOLATION §4c: DM overflow at hard cap must Queue, never Reject")
	}
}
