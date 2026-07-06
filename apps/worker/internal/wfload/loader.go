// Package wfload bridges persisted workflows (Postgres) to the compiler's
// input shape (libs/workflow.PersistedWorkflow) and back out to the
// workflow_run audit table. It is apps/worker-only glue — the compiler and
// engine themselves stay in libs/workflow (ADR-004 §4.1).
package wfload

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
	"github.com/zosmed/zosmed/libs/workflow"
)

// LoadedWorkflow pairs a compiler-ready PersistedWorkflow with the workflow's
// display name, which PersistedWorkflow itself does not carry (ADR-004 R4:
// workflow_run snapshots workflow_name, not a node reference).
type LoadedWorkflow struct {
	Name string
	PWF  workflow.PersistedWorkflow
}

// Loader reads `live` workflows (+ their nodes/edges) for one account.
type Loader struct {
	db *dbgen.Queries
}

// NewLoader returns a Loader backed by db.
func NewLoader(db *dbgen.Queries) *Loader {
	return &Loader{db: db}
}

// LoadLive returns every `live` workflow belonging to accountID, fully
// hydrated with its nodes and edges, ready for Compiler.Compile.
//
// TODO(ADR-004 R1 — optimisation ticket): this re-reads and the caller
// re-compiles on every single event (compile-per-event). Fine for MVP
// volume; revisit with a (account_id, version) compiled-engine cache if
// comment throughput grows.
func (l *Loader) LoadLive(ctx context.Context, accountID pgtype.UUID) ([]LoadedWorkflow, error) {
	wfs, err := l.db.ListLiveWorkflowsByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("wfload: list live workflows: %w", err)
	}

	out := make([]LoadedWorkflow, 0, len(wfs))
	for _, w := range wfs {
		nodeRows, err := l.db.ListNodesByWorkflow(ctx, w.ID)
		if err != nil {
			return nil, fmt.Errorf("wfload: list nodes for workflow %s: %w", uuidx.Format(w.ID), err)
		}
		edgeRows, err := l.db.ListEdgesByWorkflow(ctx, w.ID)
		if err != nil {
			return nil, fmt.Errorf("wfload: list edges for workflow %s: %w", uuidx.Format(w.ID), err)
		}

		pwf := workflow.PersistedWorkflow{
			ID:    uuidx.Format(w.ID),
			Nodes: make([]workflow.PersistedNode, 0, len(nodeRows)),
			Edges: make([]workflow.PersistedEdge, 0, len(edgeRows)),
		}
		for _, n := range nodeRows {
			pwf.Nodes = append(pwf.Nodes, workflow.PersistedNode{
				ID:        uuidx.Format(n.ID),
				Category:  workflow.NodeKind(n.Category),
				NodeType:  n.NodeType,
				Config:    json.RawMessage(n.Config),
				PositionX: int(n.PositionX),
				CreatedAt: n.CreatedAt.Time,
			})
		}
		for _, e := range edgeRows {
			pwf.Edges = append(pwf.Edges, workflow.PersistedEdge{
				FromNodeID: uuidx.Format(e.FromNodeID),
				ToNodeID:   uuidx.Format(e.ToNodeID),
			})
		}

		out = append(out, LoadedWorkflow{Name: w.Name, PWF: pwf})
	}
	return out, nil
}
