package workflow_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// ── stubs shared across compile_test.go cases ────────────────────────────────

type alwaysTrigger struct{}

func (alwaysTrigger) Match(_ context.Context, _ workflow.Event) bool { return true }

// configFilter allows only when rc.Event.Text equals its configured keyword —
// used to prove two nodes of the same node_type with different config
// compile into two independently-behaving instances.
type configFilter struct{ keyword string }

func (f configFilter) Allow(_ context.Context, rc *workflow.RunContext) (bool, error) {
	return rc.Event.Text == f.keyword, nil
}

// orderAction appends its configured name to rc.Vars["order"] — used to
// observe the sequence WorkflowDef.ActionKeys actually runs in.
type orderAction struct{ name string }

func (a orderAction) Execute(_ context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	rc.Vars["order"] += a.name
	return workflow.ActionResult{Detail: a.name}, nil
}

func triggerFactory() workflow.Factory {
	return workflow.Factory{
		Category: workflow.KindTrigger,
		Build:    func(_ json.RawMessage) (any, error) { return alwaysTrigger{}, nil },
	}
}

func configFilterFactory() workflow.Factory {
	return workflow.Factory{
		Category: workflow.KindFilter,
		Build: func(cfg json.RawMessage) (any, error) {
			var c struct {
				Keyword string `json:"keyword"`
			}
			if err := json.Unmarshal(cfg, &c); err != nil {
				return nil, err
			}
			return configFilter{keyword: c.Keyword}, nil
		},
	}
}

func orderActionFactory() workflow.Factory {
	return workflow.Factory{
		Category: workflow.KindAction,
		Build: func(cfg json.RawMessage) (any, error) {
			var c struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(cfg, &c); err != nil {
				return nil, err
			}
			return orderAction{name: c.Name}, nil
		},
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCompiler_TwoInstancesDistinctConfig(t *testing.T) {
	fmap := workflow.FactoryMap{
		"trig": triggerFactory(),
		"kw":   configFilterFactory(),
	}
	c := workflow.NewCompiler(fmap)

	pwf := workflow.PersistedWorkflow{
		ID: "wf1",
		Nodes: []workflow.PersistedNode{
			{ID: "n1", Category: workflow.KindTrigger, NodeType: "trig"},
			{ID: "n2", Category: workflow.KindFilter, NodeType: "kw", Config: json.RawMessage(`{"keyword":"halo"}`)},
			{ID: "n3", Category: workflow.KindFilter, NodeType: "kw", Config: json.RawMessage(`{"keyword":"beda"}`)},
		},
	}

	reg, def, err := c.Compile(pwf)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if len(def.FilterKeys) != 2 {
		t.Fatalf("FilterKeys = %v, want 2 entries", def.FilterKeys)
	}

	e1, ok := reg.Lookup("n2")
	if !ok {
		t.Fatal("n2 not registered")
	}
	e2, ok := reg.Lookup("n3")
	if !ok {
		t.Fatal("n3 not registered")
	}

	rc := &workflow.RunContext{Event: workflow.Event{Text: "halo"}}
	allow1, err := e1.Filter.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("n2 Allow error: %v", err)
	}
	allow2, err := e2.Filter.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("n3 Allow error: %v", err)
	}
	if !allow1 {
		t.Error("n2 (keyword=halo) should allow text 'halo'")
	}
	if allow2 {
		t.Error("n3 (keyword=beda) should reject text 'halo' — distinct config leaked between instances")
	}
}

func TestCompiler_ActionOrderFromEdges(t *testing.T) {
	fmap := workflow.FactoryMap{
		"trig": triggerFactory(),
		"act":  orderActionFactory(),
	}
	c := workflow.NewCompiler(fmap)

	now := time.Now()
	pwf := workflow.PersistedWorkflow{
		ID: "wf2",
		Nodes: []workflow.PersistedNode{
			{ID: "t1", Category: workflow.KindTrigger, NodeType: "trig"},
			// PositionX/CreatedAt are deliberately out of the wanted order to
			// prove edges — not position — determine the sequence.
			{ID: "a-last", Category: workflow.KindAction, NodeType: "act", Config: json.RawMessage(`{"name":"C"}`), PositionX: 0, CreatedAt: now},
			{ID: "a-first", Category: workflow.KindAction, NodeType: "act", Config: json.RawMessage(`{"name":"A"}`), PositionX: 100, CreatedAt: now.Add(time.Second)},
			{ID: "a-mid", Category: workflow.KindAction, NodeType: "act", Config: json.RawMessage(`{"name":"B"}`), PositionX: 50, CreatedAt: now.Add(2 * time.Second)},
		},
		Edges: []workflow.PersistedEdge{
			{FromNodeID: "a-first", ToNodeID: "a-mid"},
			{FromNodeID: "a-mid", ToNodeID: "a-last"},
		},
	}

	_, def, err := c.Compile(pwf)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	want := []string{"a-first", "a-mid", "a-last"}
	if len(def.ActionKeys) != len(want) {
		t.Fatalf("ActionKeys = %v, want %v", def.ActionKeys, want)
	}
	for i, k := range want {
		if def.ActionKeys[i] != k {
			t.Errorf("ActionKeys = %v, want %v", def.ActionKeys, want)
			break
		}
	}
}

func TestCompiler_FallbackOrderWithoutEdges(t *testing.T) {
	fmap := workflow.FactoryMap{
		"trig": triggerFactory(),
		"act":  orderActionFactory(),
	}
	c := workflow.NewCompiler(fmap)

	now := time.Now()
	pwf := workflow.PersistedWorkflow{
		ID: "wf3",
		Nodes: []workflow.PersistedNode{
			{ID: "t1", Category: workflow.KindTrigger, NodeType: "trig"},
			{ID: "a2", Category: workflow.KindAction, NodeType: "act", Config: json.RawMessage(`{"name":"2"}`), PositionX: 2, CreatedAt: now},
			{ID: "a1", Category: workflow.KindAction, NodeType: "act", Config: json.RawMessage(`{"name":"1"}`), PositionX: 1, CreatedAt: now},
		},
	}

	_, def, err := c.Compile(pwf)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if len(def.ActionKeys) != 2 || def.ActionKeys[0] != "a1" || def.ActionKeys[1] != "a2" {
		t.Errorf("ActionKeys = %v, want [a1 a2] (position_x fallback, no edges)", def.ActionKeys)
	}
}

func TestCompiler_UnknownNodeType(t *testing.T) {
	c := workflow.NewCompiler(workflow.FactoryMap{})
	pwf := workflow.PersistedWorkflow{
		ID: "wf4",
		Nodes: []workflow.PersistedNode{
			{ID: "n1", Category: workflow.KindTrigger, NodeType: "does-not-exist"},
		},
	}
	if _, _, err := c.Compile(pwf); err == nil {
		t.Fatal("expected error for unknown node_type, got nil")
	}
}

func TestCompiler_EndToEndRun(t *testing.T) {
	// Compile a tiny workflow and actually run it through the UNCHANGED engine
	// to prove the compiled Registry/WorkflowDef are wire-compatible.
	fmap := workflow.FactoryMap{
		"trig": triggerFactory(),
		"kw":   configFilterFactory(),
		"act":  orderActionFactory(),
	}
	c := workflow.NewCompiler(fmap)

	pwf := workflow.PersistedWorkflow{
		ID: "wf5",
		Nodes: []workflow.PersistedNode{
			{ID: "n-trig", Category: workflow.KindTrigger, NodeType: "trig"},
			{ID: "n-filter", Category: workflow.KindFilter, NodeType: "kw", Config: json.RawMessage(`{"keyword":"halo"}`)},
			{ID: "n-act", Category: workflow.KindAction, NodeType: "act", Config: json.RawMessage(`{"name":"sent"}`)},
		},
	}

	reg, def, err := c.Compile(pwf)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	eng := workflow.NewEngine(reg, []workflow.WorkflowDef{def})
	res, err := eng.Run(context.Background(), workflow.Event{Source: workflow.SourceComment, Text: "halo"}, stubSender{}, stubGater{})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !res.Triggered || !res.FilterPassed {
		t.Fatalf("expected triggered+filterPassed, got %+v", res)
	}
}

// stubSender/stubGater satisfy workflow.Sender/Gater for the end-to-end test
// (this compiled workflow's action doesn't call either).
type stubSender struct{}

func (stubSender) ReplyToComment(_ context.Context, _, _ string) error      { return nil }
func (stubSender) SendPrivateReply(_ context.Context, _, _, _ string) error { return nil }
func (stubSender) SendDM(_ context.Context, _, _, _ string) error          { return nil }

type stubGater struct{}

func (stubGater) Allow(_ context.Context, _ workflow.OutboundReq) (workflow.Decision, error) {
	return workflow.Decision{Action: workflow.DecisionAllow}, nil
}
