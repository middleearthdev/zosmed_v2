package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zosmed/zosmed/libs/workflow"
)

// postSelectionConfig is the config shape for NodeTypePostSelection
// (CLAUDE.md §7 "Post selection: filter media_id dari payload"; ADR-005 §2.1).
// Empty MediaIDs is fully permissive — every post/Reel passes.
type postSelectionConfig struct {
	MediaIDs []string `json:"mediaIds,omitempty"`
}

// postSelectionFilter allows the run to continue only when rc.Event.MediaID
// is one of the configured posts/Reels.
//
// Not a duplicate of comment-received's inline `mediaId` option
// (trigger_comment.go, §12a-4): the trigger filters at the entry point for
// the common single-post case, while this filter node lets a workflow
// compose post selection with other filters mid-chain and select MULTIPLE
// posts/Reels.
type postSelectionFilter struct {
	mediaIDs map[string]struct{}
}

func (f *postSelectionFilter) Allow(_ context.Context, rc *workflow.RunContext) (bool, error) {
	if len(f.mediaIDs) == 0 {
		return true, nil // permissive default (ADR-005 §2.1)
	}
	_, ok := f.mediaIDs[rc.Event.MediaID]
	return ok, nil
}

// BuildPostSelection is the Factory.Build func for NodeTypePostSelection.
func BuildPostSelection(cfg json.RawMessage) (any, error) {
	var c postSelectionConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: post-selection: parse config: %w", err)
		}
	}
	set := make(map[string]struct{}, len(c.MediaIDs))
	for _, id := range c.MediaIDs {
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	return &postSelectionFilter{mediaIDs: set}, nil
}
