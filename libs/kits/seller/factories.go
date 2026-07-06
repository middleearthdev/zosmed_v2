package seller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zosmed/zosmed/libs/workflow"
)

// node_type identifiers for the seller kit, as they appear in the feasible
// catalog (CLAUDE.md §7, ADR-004 §5 / libs/workflow/nodes.Catalog). These
// MUST match nodes.NodeTypeCommentToOrder / nodes.NodeTypeReserveStock
// exactly since workflow_node.node_type (persisted by the builder) is looked
// up against this same string in the compiler's FactoryMap. Duplicated as
// literals (not imported) so this Kit package doesn't import the neutral
// nodes package just for two string constants (§12a-4 — not worth a shared
// dependency for this).
const (
	nodeTypeCommentToOrder = "comment-to-order"
	nodeTypeReserveStock   = "reserve-stock"
)

// reserveAndReplyAction composes reserveAction + privateReplyAction into the
// single user-facing "reserve-stock" action node (ADR-004 §5: the catalog
// exposes ONE seller action, not two). Execute runs both steps in the same
// order as the legacy fallback workflow (runner.CommentToOrderWorkflow):
// reserve stock first (populates rc.Vars), then send the gated private
// reply — so a builder graph with a single "reserve-stock" node reproduces
// ADR-001's behaviour exactly.
type reserveAndReplyAction struct {
	reserve *reserveAction
	reply   *privateReplyAction
}

func (a *reserveAndReplyAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	reserveResult, err := a.reserve.Execute(ctx, rc)
	if err != nil {
		return reserveResult, err
	}
	replyResult, err := a.reply.Execute(ctx, rc)
	if err != nil {
		return replyResult, err
	}
	return workflow.ActionResult{
		Detail: fmt.Sprintf("%s; %s", reserveResult.Detail, replyResult.Detail),
	}, nil
}

// RegisterFactories adds the seller-kit node_types to fmap so the compiler
// (libs/workflow/compile.go) can build per-workflow instances keyed by node
// UUID (ADR-004 §1/§6.1 B5). Config is unused per node — svc/waPhone/
// enqueueOutbound are bound once at startup and shared by every instance,
// same as RegisterNodes (§12a-1 DRY via the shared constructors in kit.go).
func RegisterFactories(fmap workflow.FactoryMap, svc *ReservationService, waPhone string, enqueueOutbound EnqueueOutboundFunc) {
	fmap[nodeTypeCommentToOrder] = workflow.Factory{
		Category: workflow.KindTrigger,
		Build: func(_ json.RawMessage) (any, error) {
			return newCommentTrigger(), nil
		},
	}
	fmap[nodeTypeReserveStock] = workflow.Factory{
		Category: workflow.KindAction,
		Build: func(_ json.RawMessage) (any, error) {
			return &reserveAndReplyAction{
				reserve: newReserveAction(svc, waPhone),
				reply:   newPrivateReplyAction(svc, enqueueOutbound),
			}, nil
		},
	}
}
