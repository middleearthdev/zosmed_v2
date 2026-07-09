package seller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
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

// keepCodeCommentTrigger is the factory-built variant of commentTrigger for
// the "comment-to-order" node inside a user-saved workflow (ADR-005 §3 R3).
//
// The legacy commentTrigger (kit.go) fires on Source == comment and relies on
// comment:ingest pre-screening to keep-code comments before Engine.Run. After
// the ADR-005 ingest decoupling a builder workflow receives EVERY comment on
// the account, so this trigger must itself gate on a detected keep code
// (Event.Raw[RawKeyKode], set best-effort by comment_ingest) — otherwise a
// seller's comment-to-order workflow would reserve stock on every ordinary
// comment. Same end-state as before (reserve only on keep-code comments); the
// guard just moves from the ingest pre-screen into the trigger itself.
type keepCodeCommentTrigger struct{}

func (keepCodeCommentTrigger) Match(_ context.Context, e workflow.Event) bool {
	return e.Source == workflow.SourceComment && rawString(e.Raw, RawKeyKode) != ""
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
// enqueueDeferred are bound once at startup and shared by every instance,
// same as RegisterNodes (§12a-1 DRY via the shared constructors in kit.go).
func RegisterFactories(fmap workflow.FactoryMap, svc *ReservationService, waPhone string, enqueueDeferred nodes.EnqueueDeferredFunc) {
	fmap[nodeTypeCommentToOrder] = workflow.Factory{
		Category: workflow.KindTrigger,
		Build: func(_ json.RawMessage) (any, error) {
			// R3: keep-code-guarded (not the bare Source==comment commentTrigger),
			// because decoupled ingest now delivers every comment to this workflow.
			return keepCodeCommentTrigger{}, nil
		},
	}
	fmap[nodeTypeReserveStock] = workflow.Factory{
		Category: workflow.KindAction,
		Build: func(_ json.RawMessage) (any, error) {
			return &reserveAndReplyAction{
				reserve: newReserveAction(svc, waPhone),
				reply:   newPrivateReplyAction(svc, enqueueDeferred),
			}, nil
		},
	}
}
