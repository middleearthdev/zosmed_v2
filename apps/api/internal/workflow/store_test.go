package workflow

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// newTestUUID mints a distinct pgtype.UUID for test fixtures (last two bytes
// carry n so each call yields a unique, deterministic id).
func newTestUUID(n int) pgtype.UUID {
	var b [16]byte
	b[14] = byte(n >> 8)
	b[15] = byte(n)
	return pgtype.UUID{Bytes: b, Valid: true}
}

func node(id pgtype.UUID, category, nodeType string) dbgen.WorkflowNode {
	return dbgen.WorkflowNode{ID: id, Category: category, NodeType: nodeType}
}

func edge(from, to pgtype.UUID) dbgen.WorkflowEdge {
	return dbgen.WorkflowEdge{FromNodeID: from, ToNodeID: to}
}

func TestValidateForActivate_ValidGraph(t *testing.T) {
	trigger := newTestUUID(1)
	action := newTestUUID(2)
	nodes := []dbgen.WorkflowNode{
		node(trigger, "trigger", "comment-received"),
		node(action, "action", "send-whatsapp-link"),
	}

	reason, ok := validateForActivate(nodes, nil)
	if !ok {
		t.Fatalf("expected valid graph, got reason=%q", reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason on success, got %q", reason)
	}
}

func TestValidateForActivate_NoTrigger(t *testing.T) {
	action := newTestUUID(1)
	nodes := []dbgen.WorkflowNode{
		node(action, "action", "send-whatsapp-link"),
	}
	reason, ok := validateForActivate(nodes, nil)
	if ok {
		t.Fatal("expected invalid graph (no trigger)")
	}
	if reason != ReasonNoTrigger {
		t.Errorf("reason = %q, want %q", reason, ReasonNoTrigger)
	}
}

func TestValidateForActivate_NoAction(t *testing.T) {
	trigger := newTestUUID(1)
	nodes := []dbgen.WorkflowNode{
		node(trigger, "trigger", "comment-received"),
	}
	reason, ok := validateForActivate(nodes, nil)
	if ok {
		t.Fatal("expected invalid graph (no action)")
	}
	if reason != ReasonNoAction {
		t.Errorf("reason = %q, want %q", reason, ReasonNoAction)
	}
}

func TestValidateForActivate_UnknownNodeType(t *testing.T) {
	trigger := newTestUUID(1)
	action := newTestUUID(2)
	nodes := []dbgen.WorkflowNode{
		node(trigger, "trigger", "new-follower"), // §4b DO-NOT-list — never in the catalog
		node(action, "action", "send-whatsapp-link"),
	}
	reason, ok := validateForActivate(nodes, nil)
	if ok {
		t.Fatal("expected invalid graph (unknown node_type)")
	}
	if reason != ReasonUnknownNodeType {
		t.Errorf("reason = %q, want %q", reason, ReasonUnknownNodeType)
	}
}

func TestValidateForActivate_TriggerNotRunnable(t *testing.T) {
	trigger := newTestUUID(1)
	action := newTestUUID(2)
	// ADR-006 flipped all six messaging nodes (incl. "dm-received") to
	// Runnable:true, so no genuine catalog TRIGGER entry is non-runnable
	// anymore. validateForActivate switches on the persisted row's Category
	// field (not the catalog entry's own category), so a catalog-known but
	// still non-runnable ACTION node type ("ai-reply", ADR-005/006
	// Non-Scope) tagged as category "trigger" still exercises this branch.
	nodes := []dbgen.WorkflowNode{
		node(trigger, "trigger", "ai-reply"), // catalog-known, still not runnable
		node(action, "action", "send-whatsapp-link"),
	}
	reason, ok := validateForActivate(nodes, nil)
	if ok {
		t.Fatal("expected invalid graph (trigger not runnable)")
	}
	if reason != ReasonTriggerNotRunnable {
		t.Errorf("reason = %q, want %q", reason, ReasonTriggerNotRunnable)
	}
}

func TestValidateForActivate_Cycle(t *testing.T) {
	trigger := newTestUUID(1)
	a1 := newTestUUID(2)
	a2 := newTestUUID(3)
	nodes := []dbgen.WorkflowNode{
		node(trigger, "trigger", "comment-received"),
		node(a1, "action", "send-whatsapp-link"),
		node(a2, "action", "send-whatsapp-link"),
	}
	edges := []dbgen.WorkflowEdge{
		edge(a1, a2),
		edge(a2, a1), // cycle
	}
	reason, ok := validateForActivate(nodes, edges)
	if ok {
		t.Fatal("expected invalid graph (cycle)")
	}
	if reason != ReasonCycle {
		t.Errorf("reason = %q, want %q", reason, ReasonCycle)
	}
}

func TestHasCycle_AcyclicChainIsFine(t *testing.T) {
	a := newTestUUID(1)
	b := newTestUUID(2)
	c := newTestUUID(3)
	nodes := []dbgen.WorkflowNode{
		node(a, "action", "x"),
		node(b, "action", "x"),
		node(c, "action", "x"),
	}
	edges := []dbgen.WorkflowEdge{edge(a, b), edge(b, c)}
	if hasCycle(nodes, edges) {
		t.Error("expected no cycle in a simple chain")
	}
}
