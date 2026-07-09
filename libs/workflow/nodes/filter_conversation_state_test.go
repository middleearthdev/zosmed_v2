package nodes_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

func buildConversationStateFilter(t *testing.T, cfg string) workflow.Filter {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, nil)
	built, err := fmap[nodes.NodeTypeConversationState].Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	f, ok := built.(workflow.Filter)
	if !ok {
		t.Fatalf("built value does not implement workflow.Filter: %T", built)
	}
	return f
}

func rcWithLastInteraction(last time.Time) *workflow.RunContext {
	raw := map[string]any{}
	if !last.IsZero() {
		raw["last_interaction_at"] = last
	}
	return workflow.NewRunContext(workflow.Event{Source: workflow.SourceDM, Raw: raw}, nil, nil)
}

func TestConversationState_RequireOpenDefaultTrue_OpenWindowAllows(t *testing.T) {
	f := buildConversationStateFilter(t, `{}`)
	rc := rcWithLastInteraction(time.Now().Add(-1 * time.Hour))

	allow, err := f.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allow {
		t.Fatal("expected allow: window is open (< 24h) and requireOpen defaults to true")
	}
}

func TestConversationState_RequireOpenDefaultTrue_ExpiredWindowRejects(t *testing.T) {
	f := buildConversationStateFilter(t, `{}`)
	rc := rcWithLastInteraction(time.Now().Add(-25 * time.Hour))

	allow, err := f.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if allow {
		t.Fatal("expected reject: window is 25h old (> 24h)")
	}
}

// TestConversationState_AbsentKey_CommentFlow verifies the ADR-006 R4 scenario:
// on a comment-triggered flow, Raw[last_interaction_at] is never populated, so
// the window is treated as closed. requireOpen=true (default) must reject.
func TestConversationState_AbsentKey_CommentFlow(t *testing.T) {
	f := buildConversationStateFilter(t, `{}`)
	rc := rcWithLastInteraction(time.Time{}) // absent key — mirrors comment_ingest.go

	allow, err := f.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if allow {
		t.Fatal("expected reject: last_interaction_at absent → window treated as closed")
	}
}

func TestConversationState_RequireOpenFalse_ExpiredWindowAllows(t *testing.T) {
	f := buildConversationStateFilter(t, `{"requireOpen":false}`)
	rc := rcWithLastInteraction(time.Now().Add(-48 * time.Hour))

	allow, err := f.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allow {
		t.Fatal("expected allow: requireOpen=false matches a CLOSED window")
	}
}

func TestConversationState_RequireOpenFalse_OpenWindowRejects(t *testing.T) {
	f := buildConversationStateFilter(t, `{"requireOpen":false}`)
	rc := rcWithLastInteraction(time.Now())

	allow, err := f.Allow(context.Background(), rc)
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if allow {
		t.Fatal("expected reject: requireOpen=false does not match an OPEN window")
	}
}

// TestConversationState_NoOutbound is a documentation-level guard: the filter
// must compile and run with nil Sender/Gate — it never touches either (§10:
// filters are never gated, only actions are).
func TestConversationState_NoOutbound(t *testing.T) {
	f := buildConversationStateFilter(t, `{}`)
	rc := workflow.NewRunContext(workflow.Event{
		Source: workflow.SourceDM,
		Raw:    map[string]any{"last_interaction_at": time.Now()},
	}, nil, nil) // nil Sender, nil Gate — must not panic
	if _, err := f.Allow(context.Background(), rc); err != nil {
		t.Fatalf("Allow error: %v", err)
	}
}
