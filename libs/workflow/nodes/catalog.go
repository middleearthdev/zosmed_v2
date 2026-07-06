// Package nodes implements the segment-neutral node library for the Zosmed
// workflow engine (ADR-004 §4.1, CLAUDE.md §7/§8). It knows nothing about
// keep codes, reservations, or any other Kit-specific concept — those live
// in libs/kits/<segment>. This package MUST NOT import libs/kits/* (§9
// guardrail); doing so would break the engine/Kit boundary.
package nodes

import "github.com/zosmed/zosmed/libs/workflow"

// Node type identifiers — single source of truth for the feasible node
// catalog (CLAUDE.md §7, ADR-004 §5). The frontend keeps a mirrored TS
// constant (packages/types) in sync per §12a-1; this Go file is authoritative.
const (
	NodeTypeCommentReceived = "comment-received"
	NodeTypeCommentToOrder  = "comment-to-order"
	NodeTypeDMReceived      = "dm-received"
	NodeTypeStoryReply      = "story-reply"
	NodeTypeStoryMention    = "story-mention"
	NodeTypeClickToDMAd     = "click-to-dm-ad"

	NodeTypeKeywordMatch      = "keyword-match"
	NodeTypeConversationState = "conversation-state"
	NodeTypeIntent            = "intent"
	NodeTypePostSelection     = "post-selection"
	NodeTypeTimeWindow        = "time-window"

	NodeTypeReplyComment     = "reply-comment"
	NodeTypeSendDM           = "send-dm"
	NodeTypeAIReply          = "ai-reply"
	NodeTypeSendWhatsAppLink = "send-whatsapp-link"
	NodeTypeSendTrustKit     = "send-trust-kit"
	NodeTypeReserveStock     = "reserve-stock"
	NodeTypeNotifyOptin      = "notify-optin"
	NodeTypeHandoffHuman     = "handoff-human"
	NodeTypeTagContact       = "tag-contact"
	NodeTypeOutboundWebhook  = "outbound-webhook"
)

// CatalogEntry describes one node_type in the feasible palette (CLAUDE.md
// §7). Runnable marks whether iteration 1's runtime (compiler + factory map)
// can actually execute the node; non-runnable entries are palette-only until
// their ingest path or supporting service exists (ADR-004 §Non-Scope) —
// activate-time validation rejects a workflow that depends on one of these
// as its only trigger (reason "trigger_not_runnable", §3).
type CatalogEntry struct {
	Category workflow.NodeKind
	NodeType string
	Label    string
	IconKey  string
	Runnable bool
}

// Catalog is the full feasible-node palette (CLAUDE.md §7, ADR-004 §5). It
// intentionally excludes every DO-NOT-list capability (§4b): no
// new-follower trigger, no auto-follow/unfollow, no follow-status check, no
// IG-Live viewer count or live comment stream, no mass DM blast. Both
// comment triggers are documented as post/Reel webhook `comments` events —
// never IG Live (§4b.4–5).
var Catalog = []CatalogEntry{
	{Category: workflow.KindTrigger, NodeType: NodeTypeCommentReceived, Label: "Komentar masuk (post/Reel)", IconKey: "message-square", Runnable: true},
	{Category: workflow.KindTrigger, NodeType: NodeTypeCommentToOrder, Label: "Comment-to-Order (keep/C)", IconKey: "shopping-bag", Runnable: true},
	{Category: workflow.KindTrigger, NodeType: NodeTypeDMReceived, Label: "DM masuk", IconKey: "inbox", Runnable: false},
	{Category: workflow.KindTrigger, NodeType: NodeTypeStoryReply, Label: "Balasan Story", IconKey: "reply", Runnable: false},
	{Category: workflow.KindTrigger, NodeType: NodeTypeStoryMention, Label: "Mention di Story", IconKey: "at-sign", Runnable: false},
	{Category: workflow.KindTrigger, NodeType: NodeTypeClickToDMAd, Label: "Klik iklan ke DM", IconKey: "megaphone", Runnable: false},

	{Category: workflow.KindFilter, NodeType: NodeTypeKeywordMatch, Label: "Cocokkan kata kunci", IconKey: "search", Runnable: true},
	{Category: workflow.KindFilter, NodeType: NodeTypeConversationState, Label: "Status percakapan (window 24 jam)", IconKey: "clock", Runnable: false},
	{Category: workflow.KindFilter, NodeType: NodeTypeIntent, Label: "Deteksi intent (ragu/trust)", IconKey: "help-circle", Runnable: false},
	{Category: workflow.KindFilter, NodeType: NodeTypePostSelection, Label: "Pilih post/Reel", IconKey: "image", Runnable: false},
	{Category: workflow.KindFilter, NodeType: NodeTypeTimeWindow, Label: "Jendela waktu", IconKey: "calendar", Runnable: false},

	{Category: workflow.KindAction, NodeType: NodeTypeReplyComment, Label: "Balas komentar", IconKey: "message-circle", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeSendDM, Label: "Kirim DM", IconKey: "send", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeAIReply, Label: "Balasan AI", IconKey: "sparkles", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeSendWhatsAppLink, Label: "Kirim link WhatsApp", IconKey: "whatsapp", Runnable: true},
	{Category: workflow.KindAction, NodeType: NodeTypeSendTrustKit, Label: "Kirim trust-kit", IconKey: "shield-check", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeReserveStock, Label: "Reservasi stok (keep/C)", IconKey: "package", Runnable: true},
	{Category: workflow.KindAction, NodeType: NodeTypeNotifyOptin, Label: "Notifikasi opt-in", IconKey: "bell", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeHandoffHuman, Label: "Alihkan ke admin", IconKey: "user", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeTagContact, Label: "Tandai kontak", IconKey: "tag", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeOutboundWebhook, Label: "Webhook keluar", IconKey: "webhook", Runnable: false},
}

// byNodeType indexes Catalog once at package init for O(1) Lookup.
var byNodeType = func() map[string]CatalogEntry {
	m := make(map[string]CatalogEntry, len(Catalog))
	for _, e := range Catalog {
		m[e.NodeType] = e
	}
	return m
}()

// Lookup returns the catalog entry for node_type, or (zero, false) when
// node_type is not part of the feasible catalog at all — callers (activate
// validation, §3) treat that as "unknown_node_type" and reject the save/publish.
func Lookup(nodeType string) (CatalogEntry, bool) {
	e, ok := byNodeType[nodeType]
	return e, ok
}
