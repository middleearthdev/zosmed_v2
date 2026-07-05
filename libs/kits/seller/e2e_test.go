package seller_test

// e2e_test.go — end-to-end engine + seller kit tests and §4b regression guards.
//
// Uses fake services (no Redis, no Postgres, no IG API) to validate:
//   - Comment event → engine runs seller workflow → exactly 1 private reply with wa.me link
//   - Reservation transitions to waiting-pay after reply sent
//   - commentTrigger ONLY fires for SourceComment (§4b guard)
//   - Seller kit never produces outbound for live/follower/blast sources

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/workflow"
)

// ── fake Sender ───────────────────────────────────────────────────────────────

// recordingSender implements workflow.Sender by recording all outbound calls.
// Designed to surface both double-sends and wrong-method calls explicitly.
type recordingSender struct {
	privateReplies []sentPrivateReply
	dms            []sentDM
	commentReplies []sentCommentReply
}

type sentPrivateReply struct {
	igUserID  string
	commentID string
	text      string
}
type sentDM struct {
	igUserID     string
	targetUserID string
	text         string
}
type sentCommentReply struct {
	commentID string
	text      string
}

func (s *recordingSender) SendPrivateReply(_ context.Context, igUserID, commentID, text string) error {
	s.privateReplies = append(s.privateReplies, sentPrivateReply{igUserID, commentID, text})
	return nil
}
func (s *recordingSender) SendDM(_ context.Context, igUserID, targetUserID, text string) error {
	s.dms = append(s.dms, sentDM{igUserID, targetUserID, text})
	return nil
}
func (s *recordingSender) ReplyToComment(_ context.Context, commentID, text string) error {
	s.commentReplies = append(s.commentReplies, sentCommentReply{commentID, text})
	return nil
}

// ── fake Gater ────────────────────────────────────────────────────────────────

// alwaysAllowGater implements workflow.Gater, always returning DecisionAllow.
// Used in tests where safety gate logic is not under test.
type alwaysAllowGater struct{}

func (alwaysAllowGater) Allow(_ context.Context, _ workflow.OutboundReq) (workflow.Decision, error) {
	return workflow.Decision{Action: workflow.DecisionAllow, Reason: "test-allow"}, nil
}

// rejectGater implements workflow.Gater, always returning DecisionReject.
type rejectGater struct{ reason string }

func (g rejectGater) Allow(_ context.Context, _ workflow.OutboundReq) (workflow.Decision, error) {
	return workflow.Decision{Action: workflow.DecisionReject, Reason: g.reason}, nil
}

// queueGater implements workflow.Gater, always returning DecisionQueue.
type queueGater struct{ reason string }

func (g queueGater) Allow(_ context.Context, _ workflow.OutboundReq) (workflow.Decision, error) {
	return workflow.Decision{Action: workflow.DecisionQueue, Reason: g.reason}, nil
}

// ── engine builder ────────────────────────────────────────────────────────────

// buildE2EEngine creates a workflow.Engine with seller nodes registered,
// backed by db and using the comment-to-order workflow definition.
func buildE2EEngine(t *testing.T, db *stubDB, waPhone string) (*workflow.Engine, *seller.ReservationService) {
	return buildE2EEngineWithOutbound(t, db, waPhone, nil)
}

// buildE2EEngineWithOutbound is buildE2EEngine plus an injectable outbound-retry
// enqueue func (MAJOR-2) so tests can assert Queue re-enqueues the private reply.
func buildE2EEngineWithOutbound(t *testing.T, db *stubDB, waPhone string, enqueueOutbound seller.EnqueueOutboundFunc) (*workflow.Engine, *seller.ReservationService) {
	t.Helper()
	svc := seller.NewReservationService(db, noopEnqueue)
	reg := workflow.NewRegistry()
	seller.RegisterNodes(reg, svc, waPhone, enqueueOutbound)
	def := workflow.WorkflowDef{
		ID:          "comment-to-order",
		TriggerKeys: []string{seller.NodeKeyCommentTrigger},
		ActionKeys:  []string{seller.NodeKeyReserve, seller.NodeKeyPrivateReply},
	}
	return workflow.NewEngine(reg, []workflow.WorkflowDef{def}), svc
}

// commentEvent builds a workflow.Event that simulates a keep/C comment on a
// registered catalog post, with all Raw fields expected by seller kit actions.
func commentEvent(accountID, catalogPostID, code, fromUsername string) workflow.Event {
	return workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    accountID,
		ObjectID:     "comment-e2e-001",
		MediaID:      "media-e2e-001",
		FromID:       "user-e2e-001",
		FromUsername: fromUsername,
		Text:         "keep " + code,
		Raw: map[string]any{
			seller.RawKeyCatalogPostID: catalogPostID,
			seller.RawKeyKode:          code,
			seller.RawKeyHoldSeconds:   int32(300),
			seller.RawKeyIgUserID:      "ig-biz-user-001",
			seller.RawKeyCommentAt:     time.Now(),
		},
	}
}

// ── E2E happy path ────────────────────────────────────────────────────────────

// TestE2E_HappyPath_CommentToReserveToPrivateReply exercises the complete
// comment-to-order vertical slice:
//   - Engine fires commentTrigger
//   - reserveAction creates a reservation (stubDB) and sets Vars
//   - privateReplyAction gate-checks (Allow), sends exactly 1 private reply,
//     and marks the reservation as waiting-pay
//
// COMPLIANCE AUDIT A: exactly 1 private reply per comment (no DM blast).
// COMPLIANCE AUDIT B: outbound text contains the wa.me link.
// COMPLIANCE AUDIT C: reservation transitions to waiting-pay after send.
func TestE2E_HappyPath_CommentToReserveToPrivateReply(t *testing.T) {
	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	var markWaitingPayCalled bool
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			if arg.NewStatus == dbgen.ReservationStatusWaitingPay &&
				arg.ExpectedStatus == dbgen.ReservationStatusReserved {
				markWaitingPayCalled = true
				res := reservedReservation()
				res.Status = dbgen.ReservationStatusWaitingPay
				return res, nil
			}
			return dbgen.Reservation{}, errNoRows
		},
	}

	eng, _ := buildE2EEngine(t, db, "6281234567890")
	sender := &recordingSender{}

	event := commentEvent(accountIDStr, catalogIDStr, "C1", "pembeli_budi")

	result, err := eng.Run(context.Background(), event, sender, alwaysAllowGater{})
	if err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}

	// Engine must trigger and complete all actions.
	if !result.Triggered {
		t.Error("expected Triggered=true for SourceComment event")
	}
	if !result.FilterPassed {
		t.Error("expected FilterPassed=true (no filters in workflow def)")
	}
	if result.Err != nil {
		t.Errorf("unexpected engine error in step log: %v", result.Err)
	}

	// AUDIT A: exactly 1 private reply, 0 DMs.
	if got := len(sender.privateReplies); got != 1 {
		t.Errorf("AUDIT A: expected exactly 1 private reply, got %d", got)
	}
	if got := len(sender.dms); got != 0 {
		t.Errorf("AUDIT A: expected 0 DMs (private reply includes wa.me; no separate DM blast), got %d", got)
	}
	if got := len(sender.commentReplies); got != 0 {
		t.Errorf("AUDIT A: expected 0 public comment replies from seller kit, got %d", got)
	}

	// AUDIT B: reply text must contain wa.me link, kode, and buyer name.
	if len(sender.privateReplies) == 1 {
		text := sender.privateReplies[0].text
		if !strings.Contains(text, "wa.me") {
			t.Errorf("AUDIT B: private reply text must contain wa.me link; got: %q", text)
		}
		if !strings.Contains(text, "C1") {
			t.Errorf("AUDIT B: private reply text must contain kode 'C1'; got: %q", text)
		}
		if !strings.Contains(text, "pembeli_budi") {
			t.Errorf("AUDIT B: private reply text must contain buyer name 'pembeli_budi'; got: %q", text)
		}
		if !strings.HasPrefix(sender.privateReplies[0].igUserID, "ig-") {
			t.Errorf("AUDIT B: igUserID should be from Raw.ig_user_id; got: %q",
				sender.privateReplies[0].igUserID)
		}
	}

	// AUDIT C: reservation must transition to waiting-pay after reply.
	if !markWaitingPayCalled {
		t.Error("AUDIT C: MarkWaitingPay must be called after private reply is sent (reserved → waiting-pay)")
	}
}

// TestE2E_GateQueue_ReservationStaysReserved verifies that when the gate returns
// Queue (overflow), the outbound is deferred and the reservation stays in reserved
// state (MarkWaitingPay must NOT be called).
func TestE2E_GateQueue_ReservationStaysReserved(t *testing.T) {
	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	var markWaitingPayCalled bool
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			markWaitingPayCalled = true
			return reservedReservation(), nil
		},
	}

	eng, _ := buildE2EEngine(t, db, "6281234567890")
	sender := &recordingSender{}

	event := commentEvent(accountIDStr, catalogIDStr, "keep", "buyer_queued")

	result, err := eng.Run(context.Background(), event, sender, queueGater{"dm-overflow"})
	if err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}

	if !result.Triggered {
		t.Error("expected Triggered=true even with Queue gate")
	}

	// When gate returns Queue, outbound must NOT be sent.
	if len(sender.privateReplies) != 0 {
		t.Errorf("gate=Queue: private reply must not be sent; got %d sends", len(sender.privateReplies))
	}
	// Reservation must NOT transition (stays reserved until queue retries).
	if markWaitingPayCalled {
		t.Error("gate=Queue: MarkWaitingPay must NOT be called when outbound is deferred")
	}
}

// TestE2E_GateQueue_EnqueuesOutboundRetry verifies MAJOR-2: when the gate returns
// Queue and an enqueueOutbound func is wired, the private reply is re-queued
// (carrying the reservation + reply context) instead of being dropped.
func TestE2E_GateQueue_EnqueuesOutboundRetry(t *testing.T) {
	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) { return okProduct(), nil },
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
	}

	var captured seller.OutboundRetry
	captures := 0
	enqueue := seller.EnqueueOutboundFunc(func(_ context.Context, r seller.OutboundRetry, _ time.Duration) error {
		captured = r
		captures++
		return nil
	})

	eng, _ := buildE2EEngineWithOutbound(t, db, "6281234567890", enqueue)
	sender := &recordingSender{}
	event := commentEvent(accountIDStr, catalogIDStr, "keep", "buyer_queued")

	if _, err := eng.Run(context.Background(), event, sender, queueGater{"dm-overflow"}); err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}

	if len(sender.privateReplies) != 0 {
		t.Errorf("gate=Queue: must not send directly; got %d sends", len(sender.privateReplies))
	}
	if captures != 1 {
		t.Fatalf("expected exactly 1 outbound retry enqueued, got %d", captures)
	}
	if captured.ReservationID == "" || captured.ReplyText == "" || captured.CommentID != event.ObjectID {
		t.Errorf("enqueued retry missing context: %+v", captured)
	}
}

// TestE2E_GateReject_ReservationStaysReservedUntilExpire verifies that when the
// gate returns Reject (window expired, dedupe, kill-switch), the outbound is NOT
// sent and the reservation stays reserved until the expire task fires.
func TestE2E_GateReject_ReservationStaysReserved(t *testing.T) {
	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	var markWaitingPayCalled bool
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			markWaitingPayCalled = true
			return reservedReservation(), nil
		},
	}

	eng, _ := buildE2EEngine(t, db, "6281234567890")
	sender := &recordingSender{}

	event := commentEvent(accountIDStr, catalogIDStr, "C3", "buyer_rejected")

	result, err := eng.Run(context.Background(), event, sender, rejectGater{"window-expired"})
	if err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}

	if !result.Triggered {
		t.Error("expected Triggered=true even with Reject gate")
	}
	if len(sender.privateReplies) != 0 {
		t.Errorf("gate=Reject: no private reply must be sent; got %d", len(sender.privateReplies))
	}
	if markWaitingPayCalled {
		t.Error("gate=Reject: MarkWaitingPay must NOT be called when outbound is rejected")
	}
}

// ── §4b Regression guards ─────────────────────────────────────────────────────

// TestRegression_CommentTrigger_OnlyMatchesSourceComment is the primary §4b guard.
// The seller kit's commentTrigger must ONLY fire for workflow.SourceComment events.
// Any other Source (including IG Live, new follower, auto-follow) must NEVER
// trigger the workflow, preventing outbound messages for banned event types.
func TestRegression_CommentTrigger_OnlyMatchesSourceComment(t *testing.T) {
	cases := []struct {
		source   string
		wantFire bool
		section  string
	}{
		// ── allowed ──────────────────────────────────────────────────────────────
		{workflow.SourceComment, true, "SourceComment must trigger seller kit"},
		// ── §4b DO-NOT list ──────────────────────────────────────────────────────
		{workflow.SourceDM, false, "§4b: DM source is not a comment"},
		{workflow.SourceStory, false, "§4b: story reply is not a post/Reel comment"},
		{"live", false, "§4b.5: IG Live events MUST NOT trigger seller kit"},
		{"live-comments", false, "§4b.5: IG Live comments not supported"},
		{"new-follower", false, "§4b.1: follower trigger not supported"},
		{"auto-follow", false, "§4b.2: auto-follow not supported"},
		{"follow_status", false, "§4b.3: follow status check not supported"},
		{"live_viewers", false, "§4b.4: live viewer count not supported"},
		{"blast-dm", false, "§4b.6: mass DM blast not supported"},
		{"scrape", false, "§4b.7: scraping not supported"},
		{"", false, "empty source must never trigger"},
	}

	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			db := &stubDB{
				getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
					return okProduct(), nil
				},
				decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
					return okProduct(), nil
				},
				createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
					res := reservedReservation()
					res.WaLink = arg.WaLink
					return res, nil
				},
				updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
					return reservedReservation(), nil
				},
			}

			eng, _ := buildE2EEngine(t, db, "6281234567890")
			sender := &recordingSender{}

			event := commentEvent(accountIDStr, catalogIDStr, "C1", "buyer")
			event.Source = tc.source // override source for this test case

			result, err := eng.Run(context.Background(), event, sender, alwaysAllowGater{})
			if err != nil {
				t.Fatalf("Engine.Run error: %v", err)
			}

			// Check trigger.
			if tc.wantFire && !result.Triggered {
				t.Errorf("%s: expected workflow to trigger, got Triggered=false", tc.section)
			}
			if !tc.wantFire && result.Triggered {
				t.Errorf("REGRESSION §4b: workflow triggered for source=%q — %s", tc.source, tc.section)
			}

			// No outbound must be produced for banned sources.
			if !tc.wantFire {
				total := len(sender.privateReplies) + len(sender.dms) + len(sender.commentReplies)
				if total > 0 {
					t.Errorf("REGRESSION §4b: source=%q produced %d outbound message(s) — %s",
						tc.source, total, tc.section)
				}
			}
		})
	}
}

// TestRegression_AllOutboundViaGate verifies that the seller kit's private reply
// action ALWAYS calls gate.Allow before sending — the gate call must happen even
// when the gate is configured to reject.
func TestRegression_AllOutboundViaGate(t *testing.T) {
	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	var gateCalled bool
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			return reservedReservation(), nil
		},
	}

	eng, _ := buildE2EEngine(t, db, "6281234567890")
	sender := &recordingSender{}

	// Spy gate: records that Allow was called, then rejects.
	spyGate := &spyGater{
		called: &gateCalled,
		inner:  rejectGater{"spy-reject"},
	}

	event := commentEvent(accountIDStr, catalogIDStr, "C1", "buyer_spy")

	_, err := eng.Run(context.Background(), event, sender, spyGate)
	if err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}

	if !gateCalled {
		t.Error("GUARDRAIL F: gate.Allow must be called before any outbound message (one-door guarantee)")
	}
	if len(sender.privateReplies) != 0 {
		t.Errorf("rejected gate: no outbound must be sent; got %d replies", len(sender.privateReplies))
	}
}

// spyGater wraps another Gater and records that Allow was called.
type spyGater struct {
	called *bool
	inner  workflow.Gater
}

func (s *spyGater) Allow(ctx context.Context, req workflow.OutboundReq) (workflow.Decision, error) {
	*s.called = true
	return s.inner.Allow(ctx, req)
}

// TestRegression_NoLiveOrFollowerNodeRegistered verifies that the seller kit's
// RegisterNodes call does not register any trigger or action for IG Live events
// or follower events (§4b.1, §4b.2, §4b.4, §4b.5).
//
// This tests at the registry level: look up banned node keys and confirm they
// do not exist.  Any future registration of such nodes would be caught here.
func TestRegression_NoLiveOrFollowerNodeRegistered(t *testing.T) {
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, _ dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			return reservedReservation(), nil
		},
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			return reservedReservation(), nil
		},
	}

	_, svc := buildE2EEngine(t, db, "6281234567890")
	_ = svc // svc is used to verify non-nil; actual node check is via registry

	// Rebuild registry to inspect it directly.
	reg := workflow.NewRegistry()
	seller.RegisterNodes(reg, svc, "6281234567890", nil)

	// None of these keys must be registered.
	bannedKeys := []string{
		"seller.live",
		"seller.new-follower",
		"seller.auto-follow",
		"seller.live-comment",
		"seller.follower-count",
		"seller.live-viewers",
		"seller.blast-dm",
	}

	for _, key := range bannedKeys {
		_, found := reg.Lookup(key)
		if found {
			t.Errorf("REGRESSION §4b: banned node key %q is registered in the seller kit", key)
		}
	}
}

// TestRegression_OnlyOneOutboundPerComment verifies the "one private reply per
// comment" constraint from §4c.  No additional DM may be sent from the seller
// kit after the private reply for the same comment.
func TestRegression_OnlyOneOutboundPerComment(t *testing.T) {
	accountIDStr := seller.UUIDToString(testAccountID)
	catalogIDStr := seller.UUIDToString(testCatalogPostID)

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.Status = dbgen.ReservationStatusWaitingPay
			return res, nil
		},
	}

	eng, _ := buildE2EEngine(t, db, "6281234567890")
	sender := &recordingSender{}

	event := commentEvent(accountIDStr, catalogIDStr, "C1", "buyer_one_reply")

	_, err := eng.Run(context.Background(), event, sender, alwaysAllowGater{})
	if err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}

	total := len(sender.privateReplies) + len(sender.dms) + len(sender.commentReplies)
	if total > 1 {
		t.Errorf("§4c VIOLATION: expected at most 1 outbound per comment, got %d total outbound calls "+
			"(privateReplies=%d, dms=%d, commentReplies=%d)",
			total, len(sender.privateReplies), len(sender.dms), len(sender.commentReplies))
	}

	// The one outbound must be a private reply (not a public comment reply or DM blast).
	if len(sender.privateReplies) != 1 {
		t.Errorf("expected exactly 1 private reply; got %d", len(sender.privateReplies))
	}
}
