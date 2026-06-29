package workflow

import (
	"context"
	"fmt"
)

// WorkflowDef defines a single workflow as an ordered sequence of node keys.
// One or more Triggers → zero or more Filters → one or more Actions.
// Triggers use OR semantics (any match activates).
// Filters use AND semantics (all must pass for actions to run).
type WorkflowDef struct {
	// ID is a stable identifier used in RunResult and run logs.
	ID string
	// TriggerKeys are keys of KindTrigger nodes; any match activates the workflow.
	TriggerKeys []string
	// FilterKeys are keys of KindFilter nodes; all must allow for actions to run.
	FilterKeys []string
	// ActionKeys are keys of KindAction nodes; executed in order when filters pass.
	ActionKeys []string
}

// RunResult summarises the outcome of Engine.Run for one WorkflowDef evaluation.
type RunResult struct {
	// WorkflowID is the ID of the workflow that was evaluated.
	WorkflowID string
	// Triggered is true when at least one trigger matched the event.
	Triggered bool
	// FilterPassed is true when all filters allowed the run to continue.
	FilterPassed bool
	// Steps is the ordered log of every node evaluated during this run.
	Steps []StepLog
	// Err is the first terminal error encountered, if any.
	Err error
}

// Engine evaluates Events against a set of WorkflowDefs using the registered nodes.
// It is segment-neutral: it has no knowledge of keep codes, reservations, or Kit logic.
// Kit code registers nodes into the Registry before the engine is used.
type Engine struct {
	registry  *Registry
	workflows []WorkflowDef
}

// NewEngine returns an Engine that will evaluate the given workflows using registry.
func NewEngine(registry *Registry, workflows []WorkflowDef) *Engine {
	return &Engine{registry: registry, workflows: workflows}
}

// Run evaluates all registered workflows against event and executes the first
// whose triggers match and whose filters all pass.
// sender and gate are injected per-call so each IG account gets the right client.
func (e *Engine) Run(ctx context.Context, event Event, sender Sender, gate Gater) (RunResult, error) {
	rc := NewRunContext(event, sender, gate)

	for _, wf := range e.workflows {
		result, matched, err := e.runWorkflow(ctx, wf, rc)
		if err != nil {
			return result, fmt.Errorf("workflow %s: %w", wf.ID, err)
		}
		if matched {
			return result, nil
		}
	}

	// No workflow matched this event.
	return RunResult{Steps: rc.Steps}, nil
}

// runWorkflow evaluates a single WorkflowDef. It returns (result, matched, err).
// matched is true even when a filter rejects — it means "triggers fired but filter blocked".
func (e *Engine) runWorkflow(ctx context.Context, wf WorkflowDef, rc *RunContext) (RunResult, bool, error) {
	result := RunResult{WorkflowID: wf.ID}

	// Evaluate triggers — OR semantics: any match is sufficient.
	triggered := false
	for _, key := range wf.TriggerKeys {
		entry, ok := e.registry.Lookup(key)
		if !ok {
			return result, false, fmt.Errorf("trigger node %q not found in registry", key)
		}
		if entry.Kind != KindTrigger {
			return result, false, fmt.Errorf("node %q is not a trigger (kind=%s)", key, entry.Kind)
		}
		if entry.Trigger.Match(ctx, rc.Event) {
			triggered = true
			rc.AddStep(StepLog{NodeKey: key, Kind: KindTrigger, Status: StepOK})
		} else {
			rc.AddStep(StepLog{NodeKey: key, Kind: KindTrigger, Status: StepSkipped, Detail: "no match"})
		}
	}
	result.Steps = rc.Steps
	if !triggered {
		return result, false, nil
	}
	result.Triggered = true

	// Evaluate filters — AND semantics: all must pass.
	for _, key := range wf.FilterKeys {
		entry, ok := e.registry.Lookup(key)
		if !ok {
			return result, true, fmt.Errorf("filter node %q not found in registry", key)
		}
		if entry.Kind != KindFilter {
			return result, true, fmt.Errorf("node %q is not a filter (kind=%s)", key, entry.Kind)
		}
		allow, err := entry.Filter.Allow(ctx, rc)
		if err != nil {
			rc.AddStep(StepLog{NodeKey: key, Kind: KindFilter, Status: StepError, Detail: err.Error()})
			result.Steps = rc.Steps
			result.Err = err
			return result, true, nil
		}
		if !allow {
			rc.AddStep(StepLog{NodeKey: key, Kind: KindFilter, Status: StepSkipped, Detail: "filter rejected"})
			result.Steps = rc.Steps
			return result, true, nil
		}
		rc.AddStep(StepLog{NodeKey: key, Kind: KindFilter, Status: StepOK})
	}
	result.FilterPassed = true

	// Execute actions in order.
	for _, key := range wf.ActionKeys {
		entry, ok := e.registry.Lookup(key)
		if !ok {
			return result, true, fmt.Errorf("action node %q not found in registry", key)
		}
		if entry.Kind != KindAction {
			return result, true, fmt.Errorf("node %q is not an action (kind=%s)", key, entry.Kind)
		}
		ar, err := entry.Action.Execute(ctx, rc)
		if err != nil {
			rc.AddStep(StepLog{NodeKey: key, Kind: KindAction, Status: StepError, Detail: err.Error()})
			result.Steps = rc.Steps
			result.Err = err
			return result, true, nil
		}
		rc.AddStep(StepLog{NodeKey: key, Kind: KindAction, Status: StepOK, Detail: ar.Detail})
	}

	result.Steps = rc.Steps
	return result, true, nil
}
