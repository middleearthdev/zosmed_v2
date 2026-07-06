package wfload

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/runlog"
	"github.com/zosmed/zosmed/libs/workflow"
)

// RunMeta carries the request-scoped context comment_ingest.go has that
// workflow.RunResult itself does not track (ADR-004 §2.4 workflow_run columns).
type RunMeta struct {
	AccountID pgtype.UUID
	// WorkflowID is nil for the transitional fallback built-in comment-to-order
	// workflow (ADR-004 R3/R4 — it has no persisted workflow row).
	WorkflowID     *pgtype.UUID
	WorkflowName   string
	TriggerSource  string // "comment" | "dm" | "story"
	TriggerSummary string
	ObjectID       string
	DurationMs     int32
}

// RunStore persists one workflow_run row per Engine.Run outcome the caller
// decides is worth recording. ADR-004 R2: comment_ingest.go only calls
// Insert when res.Triggered is true — non-matching events are logged via
// slog only, keeping the audit table free of noise.
type RunStore struct {
	db *dbgen.Queries
}

// NewRunStore returns a RunStore backed by db.
func NewRunStore(db *dbgen.Queries) *RunStore {
	return &RunStore{db: db}
}

// Insert writes res as one workflow_run row. Status mapping (ADR-004 §2.4):
//   - res.Err != nil        -> "failed"
//   - res.Triggered, no err -> "success"
//   - !res.Triggered        -> "skipped" (defined for completeness; callers
//     following R2 generally won't reach this branch)
func (s *RunStore) Insert(ctx context.Context, res workflow.RunResult, meta RunMeta) error {
	status := "success"
	errText := ""
	switch {
	case res.Err != nil:
		status = "failed"
		errText = res.Err.Error()
	case !res.Triggered:
		status = "skipped"
	}

	stepsJSON, err := json.Marshal(toStepDTOs(res.Steps))
	if err != nil {
		return fmt.Errorf("wfload: marshal steps: %w", err)
	}

	var workflowID pgtype.UUID
	if meta.WorkflowID != nil {
		workflowID = *meta.WorkflowID
	}

	if _, err := s.db.InsertRun(ctx, dbgen.InsertRunParams{
		WorkflowID:     workflowID,
		WorkflowName:   meta.WorkflowName,
		AccountID:      meta.AccountID,
		TriggerSource:  meta.TriggerSource,
		TriggerSummary: meta.TriggerSummary,
		ObjectID:       meta.ObjectID,
		Status:         status,
		Triggered:      res.Triggered,
		FilterPassed:   res.FilterPassed,
		Steps:          stepsJSON,
		Error:          errText,
		DurationMs:     meta.DurationMs,
	}); err != nil {
		return fmt.Errorf("wfload: insert run: %w", err)
	}
	return nil
}

// toStepDTOs converts engine step logs to the JSON-tagged shape persisted in
// Postgres and consumed by apps/api/internal/workflow (single shape, §12a-1).
func toStepDTOs(steps []workflow.StepLog) []runlog.StepDTO {
	out := make([]runlog.StepDTO, 0, len(steps))
	for _, st := range steps {
		out = append(out, runlog.StepDTO{
			NodeKey: st.NodeKey,
			Kind:    string(st.Kind),
			Status:  string(st.Status),
			Detail:  st.Detail,
		})
	}
	return out
}
