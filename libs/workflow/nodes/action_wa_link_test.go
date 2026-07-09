package nodes_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

// recordingGater records the OutboundReq it was called with and always
// returns the configured Decision — used to prove the gate is consulted
// BEFORE the sender (§10 one-door guardrail).
type recordingGater struct {
	decision workflow.Decision
	lastReq  workflow.OutboundReq
	called   bool
}

func (g *recordingGater) Allow(_ context.Context, req workflow.OutboundReq) (workflow.Decision, error) {
	g.called = true
	g.lastReq = req
	return g.decision, nil
}

// recordingSender records whether SendPrivateReply/SendDM was ever invoked.
// Shared across action_wa_link_test.go, action_reply_comment_test.go, and
// action_send_dm_test.go (§12a-1 DRY — one test double, not three).
type recordingSender struct {
	sendCalled bool
	sentText   string

	// dm* fields track calls to SendDM specifically (action_send_dm_test.go);
	// kept separate from sendCalled/sentText (SendPrivateReply) so a test can
	// assert exactly which method fired.
	dmSendCalled bool
	dmTargetUser string
	dmIgUserID   string
	dmSentText   string
}

func (s *recordingSender) ReplyToComment(_ context.Context, _, _ string) error { return nil }
func (s *recordingSender) SendPrivateReply(_ context.Context, _, commentID, text string) error {
	s.sendCalled = true
	s.sentText = text
	return nil
}
func (s *recordingSender) SendDM(_ context.Context, igUserID, targetUserID, text string) error {
	s.dmSendCalled = true
	s.dmIgUserID = igUserID
	s.dmTargetUser = targetUserID
	s.dmSentText = text
	return nil
}

func buildWaLinkAction(t *testing.T, cfg string) workflow.Action {
	t.Helper()
	return buildWaLinkActionWithEnqueue(t, cfg, nil)
}

// buildWaLinkActionWithEnqueue builds the send-whatsapp-link action with a
// custom enqueueDeferred func wired (ADR-007 §3.7), so tests can assert the
// DecisionQueue → outbound:send retry path.
func buildWaLinkActionWithEnqueue(t *testing.T, cfg string, enqueueDeferred nodes.EnqueueDeferredFunc) workflow.Action {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, enqueueDeferred)
	factory := fmap[nodes.NodeTypeSendWhatsAppLink]

	built, err := factory.Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	a, ok := built.(workflow.Action)
	if !ok {
		t.Fatalf("built value does not implement workflow.Action: %T", built)
	}
	return a
}

func TestSendWhatsAppLink_AllowSendsPrivateReply(t *testing.T) {
	action := buildWaLinkAction(t, `{"phone":"6281234567890"}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    "acc-1",
		ObjectID:     "comment-1",
		FromID:       "user-1",
		FromUsername: "rina",
		Raw:          map[string]any{"ig_user_id": "biz-1", "comment_at": time.Now()},
	}, sender, gater)

	res, err := action.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !gater.called {
		t.Fatal("gate was not consulted — guardrail §10 violated")
	}
	if !sender.sendCalled {
		t.Fatal("expected SendPrivateReply to be called on DecisionAllow")
	}
	if res.Detail == "" {
		t.Error("expected a non-empty ActionResult.Detail")
	}
	// wa.me link must be present in the actually-sent text.
	if !strings.Contains(sender.sentText, "wa.me/6281234567890") {
		t.Errorf("sent text = %q, want wa.me link for 6281234567890", sender.sentText)
	}
}

func TestSendWhatsAppLink_RejectSkipsSend(t *testing.T) {
	action := buildWaLinkAction(t, `{"phone":"6281234567890"}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionReject, Reason: "kill-switch aktif"}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceComment,
		FromUsername: "rina",
		ObjectID:     "comment-1",
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.sendCalled {
		t.Fatal("SendPrivateReply must NOT be called when the gate rejects (§10)")
	}
}

func TestSendWhatsAppLink_QueueSkipsSend(t *testing.T) {
	action := buildWaLinkAction(t, `{"phone":"6281234567890"}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionQueue, Reason: "overflow"}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{FromUsername: "rina", ObjectID: "comment-1"}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.sendCalled {
		t.Fatal("SendPrivateReply must NOT be called when the gate returns Queue (§10)")
	}
}

// TestSendWhatsAppLink_QueueEnqueuesDeferredOutbound is the ADR-007 #3
// generic-retry regression: on DecisionQueue, with enqueueDeferred wired, the
// action must enqueue a DeferredOutbound with Kind=private-reply and a
// Deadline = CommentAt + 7 days (§4c) instead of silently dropping the
// message.
func TestSendWhatsAppLink_QueueEnqueuesDeferredOutbound(t *testing.T) {
	var captured nodes.DeferredOutbound
	calls := 0
	enqueue := nodes.EnqueueDeferredFunc(func(_ context.Context, d nodes.DeferredOutbound, _ time.Duration) error {
		captured = d
		calls++
		return nil
	})
	action := buildWaLinkActionWithEnqueue(t, `{"phone":"6281234567890"}`, enqueue)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionQueue, Reason: "overflow"}}
	sender := &recordingSender{}
	commentAt := time.Now().Add(-1 * time.Hour)
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    "acc-1",
		ObjectID:     "comment-q1",
		MediaID:      "media-1",
		FromID:       "user-1",
		FromUsername: "rina",
		Raw:          map[string]any{"ig_user_id": "biz-1", "comment_at": commentAt},
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.sendCalled {
		t.Fatal("SendPrivateReply must NOT be called when the gate returns Queue (§10)")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 enqueueDeferred call, got %d", calls)
	}
	if captured.Kind != "private-reply" {
		t.Errorf("Kind = %q, want private-reply", captured.Kind)
	}
	if captured.ObjectID != "comment-q1" || captured.TriggerKey != "comment-q1" {
		t.Errorf("ObjectID/TriggerKey = %q/%q, want comment-q1", captured.ObjectID, captured.TriggerKey)
	}
	wantDeadline := commentAt.Add(nodes.PrivateReplyWindow)
	if !captured.Deadline.Equal(wantDeadline) {
		t.Errorf("Deadline = %v, want CommentAt+7d = %v", captured.Deadline, wantDeadline)
	}
	if !strings.Contains(captured.Text, "wa.me/6281234567890") {
		t.Errorf("captured Text = %q, want it to contain the wa.me link", captured.Text)
	}
}

func TestSendWhatsAppLink_MissingPhoneRejectedAtBuild(t *testing.T) {
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, nil)
	factory := fmap[nodes.NodeTypeSendWhatsAppLink]

	if _, err := factory.Build(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error when config.phone is missing")
	}
}
