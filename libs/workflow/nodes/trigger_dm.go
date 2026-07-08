package nodes

import (
	"context"
	"encoding/json"

	"github.com/zosmed/zosmed/libs/workflow"
)

// Event.Raw keys written by apps/worker/internal/tasks/dm_ingest.go for EVERY
// messaging event (ADR-006 §2). Duplicated as literals here (not imported
// from apps/worker) — same "shared wire-key convention, not shared logic"
// pattern as rawKeyIgUserID/rawKeyCommentAt in action_wa_link.go (§12a-4).
const (
	rawKeyEventSubtype      = "event_subtype"
	rawKeyLastInteractionAt = "last_interaction_at"
	rawKeyAdRef             = "ad_ref"
)

// Messaging event subtype values (ADR-006 §2). MUST stay identical to the
// webhook layer's MessagingSubtype* constants (apps/api/internal/webhook) —
// a shared wire-key convention across process boundaries, not shared code.
const (
	subtypeDM           = "dm"
	subtypeStoryReply   = "story-reply"
	subtypeStoryMention = "story-mention"
	subtypeAdReferral   = "ad-referral"
)

// dmReceivedTrigger fires for a plain DM event: Source==dm AND
// Raw[event_subtype]=="dm" (ADR-006 §2.1). All four messaging triggers share
// Source==dm (every event arrives via entry[].messaging[]); the subtype key
// is the sole discriminator.
type dmReceivedTrigger struct{}

func (t *dmReceivedTrigger) Match(_ context.Context, e workflow.Event) bool {
	return e.Source == workflow.SourceDM && rawString(e.Raw, rawKeyEventSubtype) == subtypeDM
}

// BuildDMReceived is the Factory.Build func for NodeTypeDMReceived.
// Config is always `{}` (CLAUDE.md §7 "DM received"; ADR-006 §2 table).
func BuildDMReceived(cfg json.RawMessage) (any, error) {
	return &dmReceivedTrigger{}, nil
}
