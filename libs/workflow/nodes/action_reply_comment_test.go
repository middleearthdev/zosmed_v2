package nodes_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
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
