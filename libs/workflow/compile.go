package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Factory builds one NodeEntry implementation from a persisted node's config.
// Category tells Compile which Register* method to call; Build must return a
// value implementing the corresponding interface (Trigger for KindTrigger,
// Filter for KindFilter, Action for KindAction) — Compile type-asserts and
// fails loudly if a factory is mis-registered against the wrong Category.
//
// Config (keywords, template text, …) is bound once at Build time via
// closure; runtime context flows separately through Event.Raw (ADR-004 §1).
type Factory struct {
	Category NodeKind
	Build    func(cfg json.RawMessage) (any, error)
}

// FactoryMap maps node_type (the feasible-catalog identifier, §7) to the
// Factory that knows how to construct it. Assembled at startup by apps/worker
// from libs/workflow/nodes (neutral) + libs/kits/* (segment-specific) —
// neither this file nor Compiler ever imports a Kit package (guardrail §9).
type FactoryMap map[string]Factory

// PersistedNode is the compiler's view of one workflow_node row. Field names
// mirror the DB columns (ADR-004 §2.2); callers (apps/worker/internal/wfload)
// convert from dbgen rows into this shape.
type PersistedNode struct {
	ID        string
	Category  NodeKind
	NodeType  string
	Config    json.RawMessage
	PositionX int
	CreatedAt time.Time
}

// PersistedEdge is the compiler's view of one workflow_edge row.
type PersistedEdge struct {
	FromNodeID string
	ToNodeID   string
}

// PersistedWorkflow is the compiler's view of one workflow + its nodes/edges,
// as loaded from Postgres for a single account (ADR-004 §1).
type PersistedWorkflow struct {
	ID    string
	Nodes []PersistedNode
	Edges []PersistedEdge
}

// Compiler maps a PersistedWorkflow to a ready-to-run (*Registry, WorkflowDef)
// pair for the UNCHANGED engine (libs/workflow/engine.go). It is segment
// neutral: it only knows FactoryMap, never keep-codes/reservations/Kit terms.
type Compiler struct {
	factories FactoryMap
}

// NewCompiler returns a Compiler backed by factories.
func NewCompiler(factories FactoryMap) *Compiler {
	return &Compiler{factories: factories}
}

// Compile builds a Registry (keyed by node.ID — a UUID string, never a
// node_type) and a WorkflowDef for pwf. Two workflows, or two nodes of the
// same node_type with different config, never collide because each instance
// is registered under its own unique node ID (ADR-004 §1).
//
// Returns an error if any node references a node_type absent from the
// factory map, or if a factory's Build return value does not implement the
// interface its declared Category requires.
func (c *Compiler) Compile(pwf PersistedWorkflow) (*Registry, WorkflowDef, error) {
	reg := NewRegistry()
	def := WorkflowDef{ID: pwf.ID}

	// nodesByCategory retains PersistedNode (not just ID) so ActionKeys can be
	// topo-sorted below using PositionX/CreatedAt as tie-breakers.
	var actionNodes []PersistedNode

	for _, n := range pwf.Nodes {
		factory, ok := c.factories[n.NodeType]
		if !ok {
			return nil, WorkflowDef{}, fmt.Errorf("workflow: compile %s: unknown node_type %q", pwf.ID, n.NodeType)
		}
		built, err := factory.Build(n.Config)
		if err != nil {
			return nil, WorkflowDef{}, fmt.Errorf("workflow: compile %s: build node %s (%s): %w", pwf.ID, n.ID, n.NodeType, err)
		}

		switch n.Category {
		case KindTrigger:
			t, ok := built.(Trigger)
			if !ok {
				return nil, WorkflowDef{}, fmt.Errorf("workflow: compile %s: node_type %q factory does not build a Trigger", pwf.ID, n.NodeType)
			}
			reg.RegisterTrigger(n.ID, t)
			def.TriggerKeys = append(def.TriggerKeys, n.ID)
		case KindFilter:
			f, ok := built.(Filter)
			if !ok {
				return nil, WorkflowDef{}, fmt.Errorf("workflow: compile %s: node_type %q factory does not build a Filter", pwf.ID, n.NodeType)
			}
			reg.RegisterFilter(n.ID, f)
			def.FilterKeys = append(def.FilterKeys, n.ID)
		case KindAction:
			a, ok := built.(Action)
			if !ok {
				return nil, WorkflowDef{}, fmt.Errorf("workflow: compile %s: node_type %q factory does not build an Action", pwf.ID, n.NodeType)
			}
			reg.RegisterAction(n.ID, a)
			actionNodes = append(actionNodes, n)
		default:
			return nil, WorkflowDef{}, fmt.Errorf("workflow: compile %s: node %s has unknown category %q", pwf.ID, n.ID, n.Category)
		}
	}

	def.ActionKeys = topoSortActions(actionNodes, pwf.Edges)
	return reg, def, nil
}

// topoSortActions orders action node IDs using Kahn's algorithm over the
// edges restricted to action→action pairs (ADR-004 §4.3). Ties (nodes with
// no ordering edge between them) are broken by PositionX then CreatedAt so
// output is deterministic. If a cycle is present among actions, the
// remaining un-orderable nodes are appended in the same tie-break order
// rather than erroring — activate-time validation (apps/api) is the
// authoritative place a cyclic graph is rejected (§3 reason "cycle"); this
// fallback only protects a `live` workflow that somehow bypassed that check.
func topoSortActions(nodes []PersistedNode, edges []PersistedEdge) []string {
	if len(nodes) == 0 {
		return nil
	}

	byID := make(map[string]PersistedNode, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
	}

	indegree := make(map[string]int, len(nodes))
	adj := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		indegree[n.ID] = 0
	}
	for _, e := range edges {
		if _, fromIsAction := byID[e.FromNodeID]; !fromIsAction {
			continue
		}
		if _, toIsAction := byID[e.ToNodeID]; !toIsAction {
			continue
		}
		adj[e.FromNodeID] = append(adj[e.FromNodeID], e.ToNodeID)
		indegree[e.ToNodeID]++
	}

	tieBreak := func(ids []string) {
		sort.Slice(ids, func(i, j int) bool {
			a, b := byID[ids[i]], byID[ids[j]]
			if a.PositionX != b.PositionX {
				return a.PositionX < b.PositionX
			}
			if !a.CreatedAt.Equal(b.CreatedAt) {
				return a.CreatedAt.Before(b.CreatedAt)
			}
			return a.ID < b.ID
		})
	}

	var ready []string
	for id, deg := range indegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}
	tieBreak(ready)

	var order []string
	visited := make(map[string]bool, len(nodes))
	for len(ready) > 0 {
		// Pop the front (deterministic order from tieBreak above / re-sort below).
		id := ready[0]
		ready = ready[1:]
		if visited[id] {
			continue
		}
		visited[id] = true
		order = append(order, id)

		var freed []string
		for _, next := range adj[id] {
			indegree[next]--
			if indegree[next] == 0 {
				freed = append(freed, next)
			}
		}
		tieBreak(freed)
		ready = append(ready, freed...)
		tieBreak(ready)
	}

	// Cycle fallback: any node not yet visited (indegree never reached 0) is
	// appended in tie-break order (see doc comment above).
	if len(order) < len(nodes) {
		var remaining []string
		for _, n := range nodes {
			if !visited[n.ID] {
				remaining = append(remaining, n.ID)
			}
		}
		tieBreak(remaining)
		order = append(order, remaining...)
	}

	return order
}
