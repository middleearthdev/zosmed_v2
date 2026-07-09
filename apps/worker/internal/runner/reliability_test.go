package runner

// reliability_test.go — ADR-007 §5 scenarios 9 & 10, exercised at the
// composition-root level (this package is the only place allowed to import
// BOTH libs/workflow/nodes and libs/safety — see runner.go's package doc and
// the DoD guardrail "libs/workflow/nodes/* tak impor libs/kits/*/libs/safety").
//
// A fake workflow.Gater cannot prove the Kind-scoped dedupe key fix (ADR-007
// §2.3a) — that fix lives entirely inside libs/safety's Redis key format. So
// these tests wire the REAL safety.Gate (miniredis-backed, same pattern as
// libs/safety/gate_test.go) through the SAME gateAdapter runner.New() uses in
// production, running the actual neutral action nodes
// (libs/workflow/nodes) against it via a plain workflow.Engine — no Postgres,
// no asynq, no HTTP: exactly the boundary these two scenarios need.
//
// White-box (package runner, not runner_test) so the tests can reuse the
// unexported gateAdapter type instead of duplicating its ~15-line mapping
// logic (§12a-1 DRY) — it's the one already used by runner.New() and covered
// implicitly by every other runner-level test.

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/zosmed/zosmed/libs/safety"
	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

// newTestGater spins up an in-process miniredis and returns the real
// safety.Gate wrapped in the production gateAdapter — a workflow.Gater ready
// to be handed to workflow.Engine.Run.
func newTestGater(t *testing.T) workflow.Gater {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return &gateAdapter{g: safety.New(rdb)}
}

// reliabilitySender records every Sender method invocation — used to prove
// exactly how many outbound messages actually fired.
type reliabilitySender struct {
	replyToComment   int
	sendPrivateReply int
	sendDM           int
}

func (s *reliabilitySender) ReplyToComment(_ context.Context, _, _ string) error {
	s.replyToComment++
	return nil
}
func (s *reliabilitySender) SendPrivateReply(_ context.Context, _, _, _ string) error {
	s.sendPrivateReply++
	return nil
}
func (s *reliabilitySender) SendDM(_ context.Context, _, _, _ string) error {
	s.sendDM++
	return nil
}

// buildReplyAndWaLinkEngine wires the exact two-action neutral shape ADR-007
// §0's collision-bug narrative describes: comment-received → [reply-comment,
// send-whatsapp-link]. enqueueDeferred is nil — these tests only exercise the
// Allow/Reject paths (Queue/retry is covered elsewhere, Tahap C).
func buildReplyAndWaLinkEngine(t *testing.T) *workflow.Engine {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, nil)

	reg := workflow.NewRegistry()

	trig, err := fmap[nodes.NodeTypeCommentReceived].Build(nil)
	if err != nil {
		t.Fatalf("build comment-received: %v", err)
	}
	reg.RegisterTrigger("trig-1", trig.(workflow.Trigger))

	replyAction, err := fmap[nodes.NodeTypeReplyComment].Build(nil)
	if err != nil {
		t.Fatalf("build reply-comment: %v", err)
	}
	reg.RegisterAction("reply-1", replyAction.(workflow.Action))

	waAction, err := fmap[nodes.NodeTypeSendWhatsAppLink].Build([]byte(`{"phone":"6281234567890"}`))
	if err != nil {
		t.Fatalf("build send-whatsapp-link: %v", err)
	}
	reg.RegisterAction("wa-1", waAction.(workflow.Action))

	def := workflow.WorkflowDef{
		ID:          "reply-then-walink",
		TriggerKeys: []string{"trig-1"},
		ActionKeys:  []string{"reply-1", "wa-1"},
	}
	return workflow.NewEngine(reg, []workflow.WorkflowDef{def})
}

// singleWaLinkEngine wires a single-action neutral workflow (comment-received
// → send-whatsapp-link only) — the shape scenario 10 needs to isolate
// per-kind dedupe on a re-run without a second, different-Kind action
// muddying the assertion.
func singleWaLinkEngine(t *testing.T) *workflow.Engine {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, nil)

	reg := workflow.NewRegistry()

	trig, err := fmap[nodes.NodeTypeCommentReceived].Build(nil)
	if err != nil {
		t.Fatalf("build comment-received: %v", err)
	}
	reg.RegisterTrigger("trig-1", trig.(workflow.Trigger))

	waAction, err := fmap[nodes.NodeTypeSendWhatsAppLink].Build([]byte(`{"phone":"6281234567890"}`))
	if err != nil {
		t.Fatalf("build send-whatsapp-link: %v", err)
	}
	reg.RegisterAction("wa-1", waAction.(workflow.Action))

	def := workflow.WorkflowDef{
		ID:          "walink-only",
		TriggerKeys: []string{"trig-1"},
		ActionKeys:  []string{"wa-1"},
	}
	return workflow.NewEngine(reg, []workflow.WorkflowDef{def})
}

func collisionTestEvent() workflow.Event {
	return workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    "acc-collision-1",
		ObjectID:     "comment-collision-1", // same comment → same TriggerKey for BOTH actions
		MediaID:      "media-1",
		FromID:       "user-collision-1",
		FromUsername: "rina",
		Raw:          map[string]any{"ig_user_id": "biz-1"}, // zero comment_at: window check no-ops (fine — this test targets dedupe, not window)
	}
}

// TestEngine_ReplyThenWaLink_BothOutboundSent_NoCollision is ADR-007 §5
// scenario 9: a workflow [reply-comment → send-whatsapp-link] triggered by
// ONE comment must deliver BOTH outbound messages — the comment-reply (public)
// AND the private-reply (wa-link DM) — because the safety Gate's dedupe key
// now includes Kind (ADR-007 §2.3a). Before that fix, the second action would
// wrongly see the first action's dedupe mark and get Rejected.
func TestEngine_ReplyThenWaLink_BothOutboundSent_NoCollision(t *testing.T) {
	gate := newTestGater(t)
	eng := buildReplyAndWaLinkEngine(t)
	sender := &reliabilitySender{}

	result, err := eng.Run(context.Background(), collisionTestEvent(), sender, gate)
	if err != nil {
		t.Fatalf("Engine.Run error: %v", err)
	}
	if !result.Triggered || !result.FilterPassed {
		t.Fatalf("expected Triggered=true FilterPassed=true, got %+v", result)
	}
	if result.Err != nil {
		t.Fatalf("expected no action error, got: %v", result.Err)
	}
	if sender.replyToComment != 1 {
		t.Errorf("expected ReplyToComment called once, got %d", sender.replyToComment)
	}
	if sender.sendPrivateReply != 1 {
		t.Errorf("expected SendPrivateReply called once (must NOT be dedupe-collided away by reply-comment), got %d", sender.sendPrivateReply)
	}
}

// TestEngine_RerunSingleActionWorkflow_SecondSendBlockedByDedupe is ADR-007
// §5 scenario 10: re-running "comment:ingest" for an identical event on a
// single-action workflow (simulating an asynq retry that reaches the engine a
// second time) must NOT send the outbound a second time — the Gate's
// per-(kind,account,user,trigger) dedupe Rejects the repeat, exactly as it
// would for a genuine asynq re-delivery.
func TestEngine_RerunSingleActionWorkflow_SecondSendBlockedByDedupe(t *testing.T) {
	gate := newTestGater(t)
	eng := singleWaLinkEngine(t)
	sender := &reliabilitySender{}
	event := collisionTestEvent()

	// First run ("first delivery"): must send.
	if _, err := eng.Run(context.Background(), event, sender, gate); err != nil {
		t.Fatalf("first Engine.Run error: %v", err)
	}
	if sender.sendPrivateReply != 1 {
		t.Fatalf("expected 1 send on first run, got %d", sender.sendPrivateReply)
	}

	// Second run with the IDENTICAL event ("asynq retry" of the same task):
	// must NOT send again.
	result, err := eng.Run(context.Background(), event, sender, gate)
	if err != nil {
		t.Fatalf("second (re-run) Engine.Run error: %v", err)
	}
	if !result.Triggered {
		t.Fatalf("expected the re-run to still trigger (dedupe happens inside the action), got %+v", result)
	}
	if sender.sendPrivateReply != 1 {
		t.Errorf("expected outbound NOT sent a second time (Gate dedupe Reject), got %d total sends", sender.sendPrivateReply)
	}
}
