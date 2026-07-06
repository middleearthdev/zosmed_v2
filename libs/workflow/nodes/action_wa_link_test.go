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

// recordingSender records whether SendPrivateReply was ever invoked.
type recordingSender struct {
	sendCalled bool
	sentText   string
}

func (s *recordingSender) ReplyToComment(_ context.Context, _, _ string) error { return nil }
func (s *recordingSender) SendPrivateReply(_ context.Context, _, commentID, text string) error {
	s.sendCalled = true
	s.sentText = text
	return nil
}
func (s *recordingSender) SendDM(_ context.Context, _, _, _ string) error { return nil }

func buildWaLinkAction(t *testing.T, cfg string) workflow.Action {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
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

func TestSendWhatsAppLink_MissingPhoneRejectedAtBuild(t *testing.T) {
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
	factory := fmap[nodes.NodeTypeSendWhatsAppLink]

	if _, err := factory.Build(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error when config.phone is missing")
	}
}
