package nodes

import "github.com/zosmed/zosmed/libs/workflow"

// RegisterFactories adds every runnable, segment-neutral node_type in this
// package to fmap. Called once at startup (apps/worker/internal/runner) before
// libs/kits/seller.RegisterFactories adds the segment-specific entries — order
// does not matter since keys are distinct (ADR-004 §1).
func RegisterFactories(fmap workflow.FactoryMap) {
	fmap[NodeTypeCommentReceived] = workflow.Factory{Category: workflow.KindTrigger, Build: BuildCommentReceived}
	fmap[NodeTypeKeywordMatch] = workflow.Factory{Category: workflow.KindFilter, Build: BuildKeywordMatch}
	fmap[NodeTypePostSelection] = workflow.Factory{Category: workflow.KindFilter, Build: BuildPostSelection}
	fmap[NodeTypeTimeWindow] = workflow.Factory{Category: workflow.KindFilter, Build: BuildTimeWindow}
	fmap[NodeTypeSendWhatsAppLink] = workflow.Factory{Category: workflow.KindAction, Build: BuildSendWhatsAppLink}
	fmap[NodeTypeReplyComment] = workflow.Factory{Category: workflow.KindAction, Build: BuildReplyComment}
	fmap[NodeTypeOutboundWebhook] = workflow.Factory{Category: workflow.KindAction, Build: BuildOutboundWebhook}
}
