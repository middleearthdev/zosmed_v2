package nodes

import "github.com/zosmed/zosmed/libs/workflow"

// RegisterFactories adds every runnable, segment-neutral node_type in this
// package to fmap. Called once at startup (apps/worker/internal/runner) before
// libs/kits/seller.RegisterFactories adds the segment-specific entries — order
// does not matter since keys are distinct (ADR-004 §1).
//
// enqueueDeferred is injected (ADR-007 §2.1/§3.7) so the three outbound
// action nodes (send-dm, reply-comment, send-whatsapp-link) can enqueue a
// generic outbound:send retry when the safety gate returns DecisionQueue
// instead of dropping the message. Pass nil to disable retry — DecisionQueue
// then falls back to the pre-ADR-007 report-only behaviour (used by tests
// that don't care about the retry path).
func RegisterFactories(fmap workflow.FactoryMap, enqueueDeferred EnqueueDeferredFunc) {
	fmap[NodeTypeCommentReceived] = workflow.Factory{Category: workflow.KindTrigger, Build: BuildCommentReceived}
	fmap[NodeTypeKeywordMatch] = workflow.Factory{Category: workflow.KindFilter, Build: BuildKeywordMatch}
	fmap[NodeTypePostSelection] = workflow.Factory{Category: workflow.KindFilter, Build: BuildPostSelection}
	fmap[NodeTypeTimeWindow] = workflow.Factory{Category: workflow.KindFilter, Build: BuildTimeWindow}
	fmap[NodeTypeSendWhatsAppLink] = workflow.Factory{Category: workflow.KindAction, Build: newSendWhatsAppLinkFactory(enqueueDeferred)}
	fmap[NodeTypeReplyComment] = workflow.Factory{Category: workflow.KindAction, Build: newReplyCommentFactory(enqueueDeferred)}
	fmap[NodeTypeOutboundWebhook] = workflow.Factory{Category: workflow.KindAction, Build: BuildOutboundWebhook}

	// ADR-006 §2: messaging/story ingest — six neutral nodes flipped from
	// Runnable:false to true. All Source==dm; Raw[event_subtype] discriminates.
	fmap[NodeTypeDMReceived] = workflow.Factory{Category: workflow.KindTrigger, Build: BuildDMReceived}
	fmap[NodeTypeStoryReply] = workflow.Factory{Category: workflow.KindTrigger, Build: BuildStoryReply}
	fmap[NodeTypeStoryMention] = workflow.Factory{Category: workflow.KindTrigger, Build: BuildStoryMention}
	fmap[NodeTypeClickToDMAd] = workflow.Factory{Category: workflow.KindTrigger, Build: BuildClickToDMAd}
	fmap[NodeTypeConversationState] = workflow.Factory{Category: workflow.KindFilter, Build: BuildConversationState}
	fmap[NodeTypeSendDM] = workflow.Factory{Category: workflow.KindAction, Build: newSendDMFactory(enqueueDeferred)}
}
