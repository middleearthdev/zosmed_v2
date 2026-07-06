// Package runlog defines the JSON shape persisted in workflow_run.steps
// (ADR-004 §2.4/§3). workflow.StepLog (libs/workflow/event.go) intentionally
// carries no JSON tags — the engine core is transport-neutral — so this DTO
// is the single translation point shared by the writer
// (apps/worker/internal/wfload) and the reader (apps/api/internal/workflow),
// avoiding two independently-drifting copies of the same shape (§12a-1 DRY).
package runlog

// StepDTO mirrors packages/types/src/workflow.ts RunStepDTO.
type StepDTO struct {
	NodeKey string `json:"nodeKey"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
}
