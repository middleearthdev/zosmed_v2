// Package workflow provides the HTTP transport for the workflow builder REST
// API (ADR-004 §3). It reads/writes via dbgen (sqlc-generated) and maps to
// JSON DTOs aligned with packages/types/src/domain.ts + workflow.ts.
//
// SoC (§12a-3): handler.go is transport-only; Store (store.go) owns the
// transactional save (delete+insert nodes/edges, ADR-004 R4) and never
// writes an HTTP response; mapping.go is the single translation location
// between dbgen rows and DTOs (§12a-1 DRY).
package workflow

import (
	"encoding/json"
	"time"

	"github.com/zosmed/zosmed/libs/platform/runlog"
)

// NodeKindDTO mirrors packages/types/src/domain.ts NodeKind: category is the
// node's role (trigger|filter|action, matches workflow_node.category); kind
// is the node_type string from the feasible catalog (libs/workflow/nodes.Catalog).
type NodeKindDTO struct {
	Category string `json:"category"`
	Kind     string `json:"kind"`
}

// PositionDTO mirrors packages/types/src/domain.ts WorkflowNode.position.
type PositionDTO struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WorkflowNodeDTO mirrors packages/types/src/domain.ts WorkflowNode. Label is
// derived server-side from the feasible catalog (nodes.Lookup) — it is not a
// DB column, avoiding a second source of truth for display names (§12a-1).
type WorkflowNodeDTO struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Node     NodeKindDTO     `json:"node"`
	Config   json.RawMessage `json:"config"`
	Position PositionDTO     `json:"position"`
}

// WorkflowEdgeDTO mirrors packages/types/src/domain.ts WorkflowEdge.
type WorkflowEdgeDTO struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// WorkflowDTO mirrors packages/types/src/domain.ts Workflow.
type WorkflowDTO struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Segment   string            `json:"segment"`
	Nodes     []WorkflowNodeDTO `json:"nodes"`
	Edges     []WorkflowEdgeDTO `json:"edges"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// WorkflowSummaryDTO is the payload for GET /api/v1/workflows (ADR-004 §3).
type WorkflowSummaryDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Segment   string    `json:"segment"`
	NodeCount int       `json:"nodeCount"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateWorkflowRequest is the body of POST /api/v1/workflows.
type CreateWorkflowRequest struct {
	Name    string `json:"name"`
	Segment string `json:"segment"`
}

// SaveWorkflowNodeRequest is one node in SaveWorkflowRequest.Nodes. ID may be
// empty for newly-added canvas nodes — the server always assigns a fresh
// UUID on save regardless (ADR-004 R4: save = transactional replace), so the
// incoming ID is only used to correlate SaveWorkflowEdgeRequest.From/To
// within THIS request payload, never persisted as-is.
type SaveWorkflowNodeRequest struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Node     NodeKindDTO     `json:"node"`
	Config   json.RawMessage `json:"config"`
	Position PositionDTO     `json:"position"`
}

// SaveWorkflowEdgeRequest is one edge in SaveWorkflowRequest.Edges. From/To
// reference SaveWorkflowNodeRequest.ID values from the SAME request.
type SaveWorkflowEdgeRequest struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// SaveWorkflowRequest is the body of PUT /api/v1/workflows/{id}.
type SaveWorkflowRequest struct {
	Name  string                    `json:"name"`
	Nodes []SaveWorkflowNodeRequest `json:"nodes"`
	Edges []SaveWorkflowEdgeRequest `json:"edges"`
}

// RunSummaryDTO is one row for GET /api/v1/workflows/{id}/runs and
// GET /api/v1/runs (ADR-004 §3). Steps reuses runlog.StepDTO — the exact
// shape persisted in workflow_run.steps by apps/worker/internal/wfload —
// so there is only one Go type for this shape across both apps (§12a-1 DRY).
type RunSummaryDTO struct {
	ID             string           `json:"id"`
	WorkflowID     *string          `json:"workflowId"`
	WorkflowName   string           `json:"workflowName"`
	TriggerSummary string           `json:"triggerSummary"`
	Status         string           `json:"status"`
	DurationMs     int32            `json:"durationMs"`
	Steps          []runlog.StepDTO `json:"steps"`
	At             time.Time        `json:"at"`
}
