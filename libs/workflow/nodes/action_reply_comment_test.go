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

// replyRecordingSender records public comment replies (the wa-link test's
// recordingSender no-ops ReplyToComment; this action exercises that method).
type replyRecordingSender struct {
	replyCalled bool
	commentID   string
	text        string
}

func (s *replyRecordingSender) ReplyToComment(_ context.Context, commentID, text string) error {
	s.replyCalled = true
	s.commentID = commentID
	s.text = text
	return nil
}
func (s *replyRecordingSender) SendPrivateReply(_ context.Context, _, _, _ string) error { return nil }
func (s *replyRecordingSender) SendDM(_ context.Context, _, _, _ string) error           { return nil }

func buildReplyCommentAction(t *testing.T, cfg string) workflow.Action {
	t.Helper()
	return buildReplyCommentActionWithEnqueue(t, cfg, nil)
}

// buildReplyCommentActionWithEnqueue builds the reply-comment action with a
// custom enqueueDeferred func wired (ADR-007 §3.7), so tests can assert the
// DecisionQueue → outbound:send retry path.
func buildReplyCommentActionWithEnqueue(t *testing.T, cfg string, enqueueDeferred nodes.EnqueueDeferredFunc) workflow.Action {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, enqueueDeferred)
	built, err := fmap[nodes.NodeTypeReplyComment].Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	a, ok := built.(workflow.Action)
	if !ok {
		t.Fatalf("built value does not implement workflow.Action: %T", built)
	}
	return a
}

func replyCommentEvent() workflow.Event {
	return workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    "acc-1",
		ObjectID:     "comment-1",
		MediaID:      "media-1",
		FromID:       "user-1",
		FromUsername: "rina",
	}
}

func TestReplyComment_AllowPostsPublicReply(t *testing.T) {
	action := buildReplyCommentAction(t, `{"template":"Halo {nama}, ditunggu ya!"}`)
	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &replyRecordingSender{}
	rc := workflow.NewRunContext(replyCommentEvent(), sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !gater.called {
		t.Fatal("gate was not consulted before sending — guardrail §10 violated")
	}
	// §4c: public comment reply must be metered as comment-reply, not DM.
	if gater.lastReq.Kind != "comment-reply" {
		t.Errorf("gate Kind = %q, want comment-reply", gater.lastReq.Kind)
	}
	if !sender.replyCalled {
		t.Fatal("expected ReplyToComment to be called on DecisionAllow")
	}
	if sender.commentID != "comment-1" {
		t.Errorf("reply anchored to %q, want comment-1", sender.commentID)
	}
	if !strings.Contains(sender.text, "rina") {
		t.Errorf("sent text %q should interpolate {nama}=rina", sender.text)
	}
}

func TestReplyComment_RejectSkipsSend(t *testing.T) {
	action := buildReplyCommentAction(t, `{}`)
	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionReject, Reason: "kill-switch"}}
	sender := &replyRecordingSender{}
	rc := workflow.NewRunContext(replyCommentEvent(), sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.replyCalled {
		t.Fatal("ReplyToComment must NOT be called when the gate rejects (§10)")
	}
}

func TestReplyComment_QueueSkipsSend(t *testing.T) {
	action := buildReplyCommentAction(t, `{}`)
	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionQueue, Reason: "overflow"}}
	sender := &replyRecordingSender{}
	rc := workflow.NewRunContext(replyCommentEvent(), sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.replyCalled {
		t.Fatal("ReplyToComment must NOT be called when the gate returns Queue (§10)")
	}
}

// TestReplyComment_QueueEnqueuesDeferredOutbound is the ADR-007 #3
// generic-retry regression: on DecisionQueue, with enqueueDeferred wired, the
// action must enqueue a DeferredOutbound with Kind=comment-reply and a
// Deadline = CommentAt + DeferredCommentReplyTTL (comment-reply has no §4c
// hard window, ADR-007 §2.1 point 2).
func TestReplyComment_QueueEnqueuesDeferredOutbound(t *testing.T) {
	var captured nodes.DeferredOutbound
	calls := 0
	enqueue := nodes.EnqueueDeferredFunc(func(_ context.Context, d nodes.DeferredOutbound, _ time.Duration) error {
		captured = d
		calls++
		return nil
	})
	action := buildReplyCommentActionWithEnqueue(t, `{}`, enqueue)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionQueue, Reason: "overflow"}}
	sender := &replyRecordingSender{}
	event := replyCommentEvent()
	commentAt := time.Now().Add(-10 * time.Minute)
	event.Raw = map[string]any{"ig_user_id": "biz-1", "comment_at": commentAt}
	rc := workflow.NewRunContext(event, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.replyCalled {
		t.Fatal("ReplyToComment must NOT be called when the gate returns Queue (§10)")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 enqueueDeferred call, got %d", calls)
	}
	if captured.Kind != "comment-reply" {
		t.Errorf("Kind = %q, want comment-reply", captured.Kind)
	}
	if captured.ObjectID != "comment-1" || captured.TriggerKey != "comment-1" {
		t.Errorf("ObjectID/TriggerKey = %q/%q, want comment-1", captured.ObjectID, captured.TriggerKey)
	}
	wantDeadline := commentAt.Add(nodes.DeferredCommentReplyTTL)
	if !captured.Deadline.Equal(wantDeadline) {
		t.Errorf("Deadline = %v, want CommentAt+TTL = %v", captured.Deadline, wantDeadline)
	}
}
