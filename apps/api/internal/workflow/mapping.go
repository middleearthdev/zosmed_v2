package workflow

import (
	"encoding/json"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/runlog"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

// ── mapping helpers (single location — DRY §12a-1) ───────────────────────────

// mapWorkflowDTO assembles the full Workflow DTO from a workflow row plus its
// nodes/edges (already scoped to that workflow by the caller).
func mapWorkflowDTO(wf dbgen.Workflow, nodeRows []dbgen.WorkflowNode, edgeRows []dbgen.WorkflowEdge) WorkflowDTO {
	nodeDTOs := make([]WorkflowNodeDTO, 0, len(nodeRows))
	for _, n := range nodeRows {
		nodeDTOs = append(nodeDTOs, mapNodeDTO(n))
	}
	edgeDTOs := make([]WorkflowEdgeDTO, 0, len(edgeRows))
	for _, e := range edgeRows {
		edgeDTOs = append(edgeDTOs, WorkflowEdgeDTO{
			ID:   uuidx.Format(e.ID),
			From: uuidx.Format(e.FromNodeID),
			To:   uuidx.Format(e.ToNodeID),
		})
	}
	return WorkflowDTO{
		ID:        uuidx.Format(wf.ID),
		Name:      wf.Name,
		Status:    string(wf.Status),
		Segment:   wf.Segment,
		Nodes:     nodeDTOs,
		Edges:     edgeDTOs,
		UpdatedAt: wf.UpdatedAt.Time,
	}
}

// mapNodeDTO maps one workflow_node row to its DTO. Label is looked up from
// the feasible catalog (nodes.Lookup) — the DB does not store a label
// (§12a-1: one source of truth for display names). Falls back to the raw
// node_type string if the catalog doesn't recognise it (defensive; activate
// validation is what actually enforces catalog membership, §3).
func mapNodeDTO(n dbgen.WorkflowNode) WorkflowNodeDTO {
	label := n.NodeType
	if entry, ok := nodes.Lookup(n.NodeType); ok {
		label = entry.Label
	}
	cfg := json.RawMessage(n.Config)
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	return WorkflowNodeDTO{
		ID:       uuidx.Format(n.ID),
		Label:    label,
		Node:     NodeKindDTO{Category: n.Category, Kind: n.NodeType},
		Config:   cfg,
		Position: PositionDTO{X: int(n.PositionX), Y: int(n.PositionY)},
	}
}

// mapSummaryDTO maps one ListWorkflowsByAccount row to WorkflowSummaryDTO.
func mapSummaryDTO(row dbgen.ListWorkflowsByAccountRow) WorkflowSummaryDTO {
	return WorkflowSummaryDTO{
		ID:        uuidx.Format(row.ID),
		Name:      row.Name,
		Status:    string(row.Status),
		Segment:   row.Segment,
		NodeCount: int(row.NodeCount),
		UpdatedAt: row.UpdatedAt.Time,
	}
}

// mapRunSummaryDTO maps one workflow_run row to RunSummaryDTO. Steps is
// stored as a JSONB array of runlog.StepDTO (written by
// apps/worker/internal/wfload) — unmarshal errors are tolerated as an empty
// slice since the column is NOT NULL DEFAULT '[]' and never hand-edited.
func mapRunSummaryDTO(row dbgen.WorkflowRun) RunSummaryDTO {
	var steps []runlog.StepDTO
	_ = json.Unmarshal(row.Steps, &steps)

	var workflowID *string
	if row.WorkflowID.Valid {
		s := uuidx.Format(row.WorkflowID)
		workflowID = &s
	}

	return RunSummaryDTO{
		ID:             uuidx.Format(row.ID),
		WorkflowID:     workflowID,
		WorkflowName:   row.WorkflowName,
		TriggerSummary: row.TriggerSummary,
		Status:         row.Status,
		DurationMs:     row.DurationMs,
		Steps:          steps,
		At:             row.CreatedAt.Time,
	}
}
