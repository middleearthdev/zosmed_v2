package workflow

import (
	"context"
	"fmt"
)

// NodeKind identifies the role of a node in the workflow graph.
type NodeKind string

const (
	KindTrigger NodeKind = "trigger"
	KindFilter  NodeKind = "filter"
	KindAction  NodeKind = "action"
)

// Trigger decides whether an Event should start a workflow run.
// Implementations must be safe for concurrent use.
type Trigger interface {
	// Match returns true when this event should activate the trigger.
	Match(ctx context.Context, event Event) bool
}

// Filter gates continuation of a run. All filters in a workflow must pass
// (AND semantics) for actions to execute.
type Filter interface {
	// Allow returns true when the run should continue past this filter.
	Allow(ctx context.Context, rc *RunContext) (bool, error)
}

// ActionResult is returned by Action.Execute and recorded in the run log.
type ActionResult struct {
	// Detail is a human-readable note for the run log (e.g., "reservation created").
	Detail string
}

// Action performs the side-effectful work of a workflow step.
type Action interface {
	// Execute performs the action. It may mutate rc.Vars to share state
	// with subsequent nodes (e.g., storing a wa.me link for the next action).
	Execute(ctx context.Context, rc *RunContext) (ActionResult, error)
}

// NodeEntry is a registered node in the Registry.
type NodeEntry struct {
	Key     string
	Kind    NodeKind
	Trigger Trigger // non-nil when Kind == KindTrigger
	Filter  Filter  // non-nil when Kind == KindFilter
	Action  Action  // non-nil when Kind == KindAction
}

// Registry holds all node implementations keyed by a stable string identifier.
// The engine operates on registered entries; Kit code calls Register* to inject
// its nodes without creating an import cycle (dependency inversion).
type Registry struct {
	entries map[string]NodeEntry
}

// NewRegistry returns an empty, ready-to-use Registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]NodeEntry)}
}

// RegisterTrigger registers a Trigger under key.
// Panics on duplicate key — this is a programming error caught at startup.
func (r *Registry) RegisterTrigger(key string, t Trigger) {
	r.mustRegister(NodeEntry{Key: key, Kind: KindTrigger, Trigger: t})
}

// RegisterFilter registers a Filter under key.
func (r *Registry) RegisterFilter(key string, f Filter) {
	r.mustRegister(NodeEntry{Key: key, Kind: KindFilter, Filter: f})
}

// RegisterAction registers an Action under key.
func (r *Registry) RegisterAction(key string, a Action) {
	r.mustRegister(NodeEntry{Key: key, Kind: KindAction, Action: a})
}

func (r *Registry) mustRegister(e NodeEntry) {
	if _, exists := r.entries[e.Key]; exists {
		panic(fmt.Sprintf("workflow: node key %q already registered", e.Key))
	}
	r.entries[e.Key] = e
}

// Lookup returns the entry for key, or (zero, false) if not found.
func (r *Registry) Lookup(key string) (NodeEntry, bool) {
	e, ok := r.entries[key]
	return e, ok
}
