// Package nodes implements the segment-neutral node library for the Zosmed
// workflow engine (ADR-004 §4.1, CLAUDE.md §7/§8). It knows nothing about
// keep codes, reservations, or any other Kit-specific concept — those live
// in libs/kits/<segment>. This package MUST NOT import libs/kits/* (§9
// guardrail); doing so would break the engine/Kit boundary.
package nodes

import "github.com/zosmed/zosmed/libs/workflow"

// Node type identifiers — the authoritative list of feasible node types
// (CLAUDE.md §7, ADR-004 §5). The frontend mirrors these string ids in
// packages/types (AnyNodeType) per §12a-1.
//
// NOTE (ADR-005): the *config schema* that drives the builder's inspector form
// lives only in the frontend (packages/types/src/workflow.ts) — it is a pure
// render concern. Go does NOT hold a schema; each node validates its own config
// VALUES in Factory.Build (e.g. action_outbound_webhook rejects unsafe URLs,
// filter_time_window rejects out-of-range weekdays). This split keeps the
// friendly, UI-shaped schema (time pickers, weekday toggles) out of the
// backend while value-validation stays authoritative where the value is used.
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
// §7). Runnable marks whether the current runtime (compiler + factory map)
// can actually execute the node; non-runnable entries are palette-only until
// their ingest path or supporting service exists (ADR-004 §Non-Scope,
// ADR-005 §1) — activate-time validation rejects a workflow that depends on
// one of these as its only trigger (reason "trigger_not_runnable", §3).
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
	{Category: workflow.KindTrigger, NodeType: NodeTypeDMReceived, Label: "DM masuk", IconKey: "inbox", Runnable: true},
	{Category: workflow.KindTrigger, NodeType: NodeTypeStoryReply, Label: "Balasan Story", IconKey: "reply", Runnable: true},
	{Category: workflow.KindTrigger, NodeType: NodeTypeStoryMention, Label: "Mention di Story", IconKey: "at-sign", Runnable: true},
	{Category: workflow.KindTrigger, NodeType: NodeTypeClickToDMAd, Label: "Klik iklan ke DM", IconKey: "megaphone", Runnable: true},

	{Category: workflow.KindFilter, NodeType: NodeTypeKeywordMatch, Label: "Cocokkan kata kunci", IconKey: "search", Runnable: true},
	{Category: workflow.KindFilter, NodeType: NodeTypeConversationState, Label: "Status percakapan (window 24 jam)", IconKey: "clock", Runnable: true},
	{Category: workflow.KindFilter, NodeType: NodeTypeIntent, Label: "Deteksi intent (ragu/trust)", IconKey: "help-circle", Runnable: false},
	{Category: workflow.KindFilter, NodeType: NodeTypePostSelection, Label: "Pilih post/Reel", IconKey: "image", Runnable: true},
	{Category: workflow.KindFilter, NodeType: NodeTypeTimeWindow, Label: "Jendela waktu", IconKey: "calendar", Runnable: true},

	{Category: workflow.KindAction, NodeType: NodeTypeReplyComment, Label: "Balas komentar", IconKey: "message-circle", Runnable: true},
	{Category: workflow.KindAction, NodeType: NodeTypeSendDM, Label: "Kirim DM", IconKey: "send", Runnable: true},
	{Category: workflow.KindAction, NodeType: NodeTypeAIReply, Label: "Balasan AI", IconKey: "sparkles", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeSendWhatsAppLink, Label: "Kirim link WhatsApp", IconKey: "whatsapp", Runnable: true},
	{Category: workflow.KindAction, NodeType: NodeTypeSendTrustKit, Label: "Kirim trust-kit", IconKey: "shield-check", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeReserveStock, Label: "Reservasi stok (keep/C)", IconKey: "package", Runnable: true},
	{Category: workflow.KindAction, NodeType: NodeTypeNotifyOptin, Label: "Notifikasi opt-in", IconKey: "bell", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeHandoffHuman, Label: "Alihkan ke admin", IconKey: "user", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeTagContact, Label: "Tandai kontak", IconKey: "tag", Runnable: false},
	{Category: workflow.KindAction, NodeType: NodeTypeOutboundWebhook, Label: "Webhook keluar", IconKey: "webhook", Runnable: true},
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
