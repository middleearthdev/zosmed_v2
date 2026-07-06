package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

// Sentinel errors returned by Store. handler.go translates these into the
// HTTP envelope (SoC §12a-3: store.go never writes an HTTP response).
var (
	// ErrNotFound is returned when the workflow id/account_id pair matches no row.
	ErrNotFound = errors.New("workflow: not found")
	// ErrUnknownEdgeNode is returned by Save when an edge references a node id
	// that is not present in the same request's node list.
	ErrUnknownEdgeNode = errors.New("workflow: edge references a node id not present in this request")
)

// NodeInput is the store-level shape of one node to persist (handler.go maps
// SaveWorkflowNodeRequest -> NodeInput after validating structural shape).
type NodeInput struct {
	// ClientID correlates EdgeInput.From/To to this node WITHIN one Save call.
	// It is never persisted — InsertNode always assigns a fresh server UUID
	// (ADR-004 R4: save = transactional replace, not an upsert-by-id).
	ClientID  string
	Category  string
	NodeType  string
	Config    []byte
	PositionX int
	PositionY int
}

// EdgeInput is the store-level shape of one edge to persist, referencing
// NodeInput.ClientID values from the same Save call.
type EdgeInput struct {
	From string
	To   string
}

// Store owns the transactional persistence for save (ADR-004 R4) and the
// read queries the HTTP handler needs. A single concrete Store is used
// throughout — no repository interface, since there is only one
// implementation (anti-over-abstraction §12a-4, consistent with auth.Store).
type Store struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// NewStore returns a Store backed by pool (for the Save transaction) and q
// (for plain reads that don't need a transaction).
func NewStore(pool *pgxpool.Pool, q *dbgen.Queries) *Store {
	return &Store{pool: pool, q: q}
}

// Save renames the workflow and REPLACES its entire node/edge set inside one
// transaction (ADR-004 §3/R4): delete all existing nodes+edges, then insert
// the new ones. Node ids always come back fresh from Postgres — the graph's
// node identity is not stable across saves, which is why workflow_run
// snapshots workflow_name rather than a node reference (R4).
//
// Returns ErrNotFound when (workflowID, accountID) matches no row (also
// covers "not yours" — GetWorkflowByID-style scoping via the same guarded
// UPDATE, R5). Returns ErrUnknownEdgeNode if an edge references a node id
// absent from nodeInputs.
func (s *Store) Save(
	ctx context.Context,
	workflowID, accountID pgtype.UUID,
	name string,
	nodeInputs []NodeInput,
	edgeInputs []EdgeInput,
) (dbgen.Workflow, []dbgen.WorkflowNode, []dbgen.WorkflowEdge, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: begin tx: %w", err)
	}
	// Rollback is a no-op once Commit succeeds — safe on the happy path,
	// guarantees rollback on any early return (mirrors seller.pgxTxRunner).
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	wf, err := q.UpdateWorkflowMeta(ctx, dbgen.UpdateWorkflowMetaParams{
		Name:      name,
		ID:        workflowID,
		AccountID: accountID,
	})
	if err != nil {
		if isNoRows(err) {
			return dbgen.Workflow{}, nil, nil, ErrNotFound
		}
		return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: update meta: %w", err)
	}

	if err := q.DeleteEdgesByWorkflow(ctx, workflowID); err != nil {
		return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: delete edges: %w", err)
	}
	if err := q.DeleteNodesByWorkflow(ctx, workflowID); err != nil {
		return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: delete nodes: %w", err)
	}

	idMap := make(map[string]pgtype.UUID, len(nodeInputs))
	newNodes := make([]dbgen.WorkflowNode, 0, len(nodeInputs))
	for _, n := range nodeInputs {
		row, err := q.InsertNode(ctx, dbgen.InsertNodeParams{
			WorkflowID: workflowID,
			Category:   n.Category,
			NodeType:   n.NodeType,
			Config:     n.Config,
			PositionX:  int32(n.PositionX),
			PositionY:  int32(n.PositionY),
		})
		if err != nil {
			return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: insert node: %w", err)
		}
		idMap[n.ClientID] = row.ID
		newNodes = append(newNodes, row)
	}

	newEdges := make([]dbgen.WorkflowEdge, 0, len(edgeInputs))
	for _, e := range edgeInputs {
		fromID, ok := idMap[e.From]
		if !ok {
			return dbgen.Workflow{}, nil, nil, ErrUnknownEdgeNode
		}
		toID, ok := idMap[e.To]
		if !ok {
			return dbgen.Workflow{}, nil, nil, ErrUnknownEdgeNode
		}
		row, err := q.InsertEdge(ctx, dbgen.InsertEdgeParams{
			WorkflowID: workflowID,
			FromNodeID: fromID,
			ToNodeID:   toID,
		})
		if err != nil {
			return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: insert edge: %w", err)
		}
		newEdges = append(newEdges, row)
	}

	if err := tx.Commit(ctx); err != nil {
		return dbgen.Workflow{}, nil, nil, fmt.Errorf("workflow: commit tx: %w", err)
	}
	return wf, newNodes, newEdges, nil
}

// ── activate validation (ADR-004 §3) ─────────────────────────────────────────

// Validation failure reason codes returned in the 422 envelope's error.reason
// (ADR-004 §3). Mirrored by the frontend per §12a-1.
const (
	ReasonNoTrigger          = "no_trigger"
	ReasonNoAction           = "no_action"
	ReasonUnknownNodeType    = "unknown_node_type"
	ReasonTriggerNotRunnable = "trigger_not_runnable"
	ReasonCycle              = "cycle"
)

// validateForActivate enforces ADR-004 §3's activation rules against the
// single source of truth for the feasible catalog (libs/workflow/nodes).
// Returns ("", true) when the graph is valid to publish.
//
// Scope note: only TRIGGER runnability is checked against ReasonTriggerNotRunnable,
// matching the ADR's five named reasons exactly. A workflow using a
// catalog-known but non-runnable FILTER or ACTION (e.g. "ai-reply") is not
// rejected here — Compile will fail defensively at runtime (unknown to the
// FactoryMap) and the run is skipped + logged, never sent (§10 unaffected,
// since no outbound is attempted without a successful compile). This
// mirrors the ADR's own listed scope (§0 Non-Scope: full §7 catalog is a
// roadmap, not a blocker) rather than inventing an unspecified reason code.
func validateForActivate(nodeRows []dbgen.WorkflowNode, edgeRows []dbgen.WorkflowEdge) (reason string, ok bool) {
	hasTrigger := false
	hasAction := false

	for _, n := range nodeRows {
		entry, known := nodes.Lookup(n.NodeType)
		if !known {
			return ReasonUnknownNodeType, false
		}
		switch workflow.NodeKind(n.Category) {
		case workflow.KindTrigger:
			hasTrigger = true
			if !entry.Runnable {
				return ReasonTriggerNotRunnable, false
			}
		case workflow.KindAction:
			hasAction = true
		}
	}

	if !hasTrigger {
		return ReasonNoTrigger, false
	}
	if !hasAction {
		return ReasonNoAction, false
	}
	if hasCycle(nodeRows, edgeRows) {
		return ReasonCycle, false
	}
	return "", true
}

// hasCycle reports whether the node/edge graph contains a cycle, via Kahn's
// algorithm (topological sort): if fewer nodes are visited than exist, some
// remain stuck at indegree > 0, meaning a cycle exists.
func hasCycle(nodeRows []dbgen.WorkflowNode, edgeRows []dbgen.WorkflowEdge) bool {
	ids := make(map[pgtype.UUID]bool, len(nodeRows))
	indegree := make(map[pgtype.UUID]int, len(nodeRows))
	adj := make(map[pgtype.UUID][]pgtype.UUID, len(nodeRows))
	for _, n := range nodeRows {
		ids[n.ID] = true
		indegree[n.ID] = 0
	}
	for _, e := range edgeRows {
		if !ids[e.FromNodeID] || !ids[e.ToNodeID] {
			continue
		}
		adj[e.FromNodeID] = append(adj[e.FromNodeID], e.ToNodeID)
		indegree[e.ToNodeID]++
	}

	queue := make([]pgtype.UUID, 0, len(nodeRows))
	for id, d := range indegree {
		if d == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range adj[id] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return visited != len(nodeRows)
}

// validationMessage returns the Bahasa Indonesia message for a reason code.
func validationMessage(reason string) string {
	switch reason {
	case ReasonNoTrigger:
		return "Workflow butuh minimal satu trigger"
	case ReasonNoAction:
		return "Workflow butuh minimal satu action"
	case ReasonUnknownNodeType:
		return "Ada node dengan tipe yang tidak dikenal/tidak diizinkan"
	case ReasonTriggerNotRunnable:
		return "Trigger yang dipakai belum bisa dijalankan pada iterasi ini"
	case ReasonCycle:
		return "Alur workflow membentuk siklus (cycle) — perbaiki koneksi antar node"
	default:
		return "Workflow tidak valid"
	}
}

// isNoRows returns true when err represents "no rows in result set" from pgx.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
