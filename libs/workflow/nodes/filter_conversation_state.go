package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// messagingWindowHours mirrors safety.MessagingWindowHours (§4c: 24h). Kept
// as a local literal — this neutral package MUST NOT import libs/safety
// (§9 guardrail), same pattern as replyCommentKind/timeWindow's local
// constants (§12a-4 rule of three: this is a wire-value convention, not
// shared logic).
const messagingWindowHours = 24 * time.Hour

// conversationStateConfig is the config shape for NodeTypeConversationState
// (ADR-006 §2 table). RequireOpen defaults to true when omitted: "only
// continue while the 24h messaging window with this contact is still open".
type conversationStateConfig struct {
	RequireOpen *bool `json:"requireOpen,omitempty"`
}

// conversationStateFilter is a pure server-side read of the window store
// (Raw[last_interaction_at], populated by dm_ingest.go from the `conversation`
// table — ADR-006 §2.2). It performs NO outbound call — filters never touch
// rc.Gate/rc.Sender.
//
// The filter is only meaningful on the DM/story flow, where Raw carries
// last_interaction_at. On the comment flow that key is absent, so `open`
// evaluates false — a filter configured with requireOpen=true correctly
// fails there (ADR-006 §2.2).
type conversationStateFilter struct {
	requireOpen bool
}

func (f *conversationStateFilter) Allow(_ context.Context, rc *workflow.RunContext) (bool, error) {
	last := rawTime(rc.Event.Raw, rawKeyLastInteractionAt)
	open := !last.IsZero() && time.Since(last) < messagingWindowHours
	return open == f.requireOpen, nil
}

// BuildConversationState is the Factory.Build func for NodeTypeConversationState.
func BuildConversationState(cfg json.RawMessage) (any, error) {
	var c conversationStateConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: conversation-state: parse config: %w", err)
		}
	}
	requireOpen := true
	if c.RequireOpen != nil {
		requireOpen = *c.RequireOpen
	}
	return &conversationStateFilter{requireOpen: requireOpen}, nil
}
