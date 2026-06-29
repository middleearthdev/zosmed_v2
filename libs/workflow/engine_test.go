package workflow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// --- inline stubs ---

// matchAll is a Trigger that always matches.
type matchAll struct{}

func (matchAll) Match(_ context.Context, _ workflow.Event) bool { return true }

// matchNone is a Trigger that never matches.
type matchNone struct{}

func (matchNone) Match(_ context.Context, _ workflow.Event) bool { return false }

// allowFilter always passes.
type allowFilter struct{}

func (allowFilter) Allow(_ context.Context, _ *workflow.RunContext) (bool, error) { return true, nil }

// rejectFilter always rejects.
type rejectFilter struct{}

func (rejectFilter) Allow(_ context.Context, _ *workflow.RunContext) (bool, error) {
	return false, nil
}

// errorFilter returns an error.
type errorFilter struct{}

func (errorFilter) Allow(_ context.Context, _ *workflow.RunContext) (bool, error) {
	return false, errors.New("filter exploded")
}

// noopAction does nothing and records a detail.
type noopAction struct{ detail string }

func (a noopAction) Execute(_ context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	return workflow.ActionResult{Detail: a.detail}, nil
}

// failAction returns an error.
type failAction struct{}

func (failAction) Execute(_ context.Context, _ *workflow.RunContext) (workflow.ActionResult, error) {
	return workflow.ActionResult{}, errors.New("action exploded")
}

// varAction writes a Var and returns success.
type varAction struct{ key, val string }

func (a varAction) Execute(_ context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	rc.Vars[a.key] = a.val
	return workflow.ActionResult{Detail: "var set"}, nil
}

// noopSender satisfies workflow.Sender without doing anything.
type noopSender struct{}

func (noopSender) ReplyToComment(_ context.Context, _, _ string) error          { return nil }
func (noopSender) SendPrivateReply(_ context.Context, _, _, _ string) error     { return nil }
func (noopSender) SendDM(_ context.Context, _, _, _ string) error               { return nil }

// noopGater satisfies workflow.Gater, always allowing.
type noopGater struct{}

func (noopGater) Allow(_ context.Context, _ workflow.OutboundReq) (workflow.Decision, error) {
	return workflow.Decision{Action: workflow.DecisionAllow}, nil
}

func testEvent() workflow.Event {
	return workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    "acct-1",
		ObjectID:     "comment-1",
		MediaID:      "media-1",
		FromID:       "user-1",
		FromUsername: "pelanggan_tes",
		Text:         "keep",
	}
}

func newEngine(reg *workflow.Registry, wf ...workflow.WorkflowDef) *workflow.Engine {
	return workflow.NewEngine(reg, wf)
}

// --- tests ---

func TestEngine_NoWorkflows(t *testing.T) {
	reg := workflow.NewRegistry()
	eng := newEngine(reg)
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Triggered {
		t.Error("expected Triggered=false when no workflows registered")
	}
}

func TestEngine_TriggerNoMatch(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("never", matchNone{})

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-no-match",
		TriggerKeys: []string{"never"},
	})
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Triggered {
		t.Error("expected Triggered=false")
	}
	// One step recorded for the trigger evaluation (skipped).
	if len(result.Steps) != 1 || result.Steps[0].Status != workflow.StepSkipped {
		t.Errorf("expected 1 skipped step, got %v", result.Steps)
	}
}

func TestEngine_TriggerMatch_NoFilters_ActionRuns(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})
	reg.RegisterAction("noop", noopAction{detail: "done"})

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-full",
		TriggerKeys: []string{"always"},
		ActionKeys:  []string{"noop"},
	})
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected Triggered=true")
	}
	if !result.FilterPassed {
		t.Error("expected FilterPassed=true when no filters")
	}
	// Should have 2 steps: trigger (ok) + action (ok).
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d: %v", len(result.Steps), result.Steps)
	}
	if result.Steps[1].Status != workflow.StepOK || result.Steps[1].Detail != "done" {
		t.Errorf("unexpected action step: %v", result.Steps[1])
	}
}

func TestEngine_FilterRejects(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})
	reg.RegisterFilter("block", rejectFilter{})
	reg.RegisterAction("noop", noopAction{detail: "done"})

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-filtered",
		TriggerKeys: []string{"always"},
		FilterKeys:  []string{"block"},
		ActionKeys:  []string{"noop"},
	})
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Triggered {
		t.Error("expected Triggered=true")
	}
	if result.FilterPassed {
		t.Error("expected FilterPassed=false when filter rejects")
	}
	// Steps: trigger(ok), filter(skipped). Action must NOT run.
	for _, s := range result.Steps {
		if s.Kind == workflow.KindAction {
			t.Errorf("action should not have run when filter rejected; got step: %v", s)
		}
	}
}

func TestEngine_FilterError(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})
	reg.RegisterFilter("err", errorFilter{})

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-filter-err",
		TriggerKeys: []string{"always"},
		FilterKeys:  []string{"err"},
	})
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error from Run: %v", err)
	}
	if result.Err == nil {
		t.Error("expected result.Err to be set for filter error")
	}
}

func TestEngine_ActionError(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})
	reg.RegisterAction("fail", failAction{})
	reg.RegisterAction("noop", noopAction{detail: "should-not-run"})

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-action-err",
		TriggerKeys: []string{"always"},
		ActionKeys:  []string{"fail", "noop"},
	})
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error from Run: %v", err)
	}
	if result.Err == nil {
		t.Error("expected result.Err for failing action")
	}
	// "noop" must NOT have run after "fail".
	for _, s := range result.Steps {
		if s.NodeKey == "noop" {
			t.Error("second action should not run after first action errored")
		}
	}
}

func TestEngine_VarsPassedBetweenActions(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})
	reg.RegisterAction("set-var", varAction{key: "produk", val: "Baju Ungu M"})

	var gotVars map[string]string
	reg.RegisterAction("read-var", actionFunc(func(_ context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
		gotVars = make(map[string]string, len(rc.Vars))
		for k, v := range rc.Vars {
			gotVars[k] = v
		}
		return workflow.ActionResult{Detail: "read"}, nil
	}))

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-vars",
		TriggerKeys: []string{"always"},
		ActionKeys:  []string{"set-var", "read-var"},
	})
	_, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotVars["produk"] != "Baju Ungu M" {
		t.Errorf("expected Vars[produk]=Baju Ungu M, got: %v", gotVars)
	}
}

func TestEngine_FirstWorkflowMatchStops(t *testing.T) {
	// Two workflows both match; only the first should run.
	executed := map[string]bool{}

	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})
	reg.RegisterAction("wf1-action", actionFunc(func(_ context.Context, _ *workflow.RunContext) (workflow.ActionResult, error) {
		executed["wf1"] = true
		return workflow.ActionResult{Detail: "wf1"}, nil
	}))
	reg.RegisterAction("wf2-action", actionFunc(func(_ context.Context, _ *workflow.RunContext) (workflow.ActionResult, error) {
		executed["wf2"] = true
		return workflow.ActionResult{Detail: "wf2"}, nil
	}))

	eng := newEngine(reg,
		workflow.WorkflowDef{ID: "wf1", TriggerKeys: []string{"always"}, ActionKeys: []string{"wf1-action"}},
		workflow.WorkflowDef{ID: "wf2", TriggerKeys: []string{"always"}, ActionKeys: []string{"wf2-action"}},
	)
	result, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WorkflowID != "wf1" {
		t.Errorf("expected WorkflowID=wf1, got %s", result.WorkflowID)
	}
	if executed["wf2"] {
		t.Error("second workflow should not have executed")
	}
}

func TestEngine_UnknownNodeKeyErrors(t *testing.T) {
	reg := workflow.NewRegistry()
	reg.RegisterTrigger("always", matchAll{})

	eng := newEngine(reg, workflow.WorkflowDef{
		ID:          "wf-unknown",
		TriggerKeys: []string{"always"},
		ActionKeys:  []string{"does-not-exist"},
	})
	_, err := eng.Run(context.Background(), testEvent(), noopSender{}, noopGater{})
	if err == nil {
		t.Fatal("expected error for unknown action node key")
	}
}

func TestOutboundReq_Fields(t *testing.T) {
	// Smoke-test that OutboundReq and DecisionAction constants are accessible.
	req := workflow.OutboundReq{
		AccountID:    "acct",
		Kind:         "private-reply",
		TargetUserID: "u1",
		TriggerKey:   "comment-1",
		CommentID:    "comment-1",
		CommentAt:    time.Now(),
	}
	if req.Kind != "private-reply" {
		t.Errorf("unexpected Kind: %s", req.Kind)
	}
	d := workflow.Decision{Action: workflow.DecisionAllow, Reason: "ok"}
	if d.Action != workflow.DecisionAllow {
		t.Error("DecisionAllow should be zero value")
	}
}

// actionFunc is a helper adapter so we can use an inline function as an Action.
type actionFunc func(context.Context, *workflow.RunContext) (workflow.ActionResult, error)

func (f actionFunc) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	return f(ctx, rc)
}
