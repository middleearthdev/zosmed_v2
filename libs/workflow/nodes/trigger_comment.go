package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zosmed/zosmed/libs/workflow"
)

// commentReceivedConfig is the config shape for NodeTypeCommentReceived.
// mediaId is optional — when set, the trigger only matches comments on that
// specific post/Reel (post-selection can also be modelled as a separate
// filter node, §7; this inline option covers the common single-post case
// without requiring a second node).
type commentReceivedConfig struct {
	MediaID string `json:"mediaId,omitempty"`
}

// commentReceivedTrigger fires for any event whose Source is "comment" (a
// webhook `comments` event on a post/Reel). Guardrail §4b.5: this is
// intentionally NOT wired to any IG Live surface — there is no such webhook.
type commentReceivedTrigger struct {
	mediaID string
}

func (t *commentReceivedTrigger) Match(_ context.Context, e workflow.Event) bool {
	if e.Source != workflow.SourceComment {
		return false
	}
	if t.mediaID != "" && e.MediaID != t.mediaID {
		return false
	}
	return true
}

// BuildCommentReceived is the Factory.Build func for NodeTypeCommentReceived.
func BuildCommentReceived(cfg json.RawMessage) (any, error) {
	var c commentReceivedConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: comment-received: parse config: %w", err)
		}
	}
	return &commentReceivedTrigger{mediaID: c.MediaID}, nil
}
