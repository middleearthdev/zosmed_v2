package nodes

import (
	"context"
	"encoding/json"

	"github.com/zosmed/zosmed/libs/workflow"
)

// storyReplyTrigger fires for a story-reply event: Source==dm AND
// Raw[event_subtype]=="story-reply" (ADR-006 §2.1). Story replies open the
// 24h messaging window like any other DM event (§4c).
type storyReplyTrigger struct{}

func (t *storyReplyTrigger) Match(_ context.Context, e workflow.Event) bool {
	return e.Source == workflow.SourceDM && rawString(e.Raw, rawKeyEventSubtype) == subtypeStoryReply
}

// BuildStoryReply is the Factory.Build func for NodeTypeStoryReply.
func BuildStoryReply(cfg json.RawMessage) (any, error) {
	return &storyReplyTrigger{}, nil
}

// storyMentionTrigger fires for a story-mention event: Source==dm AND
// Raw[event_subtype]=="story-mention" (ADR-006 §2.1/koreksi B0). Story
// mentions are event MESSAGING (message.attachments[].type=="story_mention")
// — NOT changes[].mentions (a different, comment/caption-level capability,
// out of scope — ADR-006 §0/§9). Per ADR-006 R3, story-mention OPENS/refreshes
// the 24h window exactly like story-reply.
type storyMentionTrigger struct{}

func (t *storyMentionTrigger) Match(_ context.Context, e workflow.Event) bool {
	return e.Source == workflow.SourceDM && rawString(e.Raw, rawKeyEventSubtype) == subtypeStoryMention
}

// BuildStoryMention is the Factory.Build func for NodeTypeStoryMention.
func BuildStoryMention(cfg json.RawMessage) (any, error) {
	return &storyMentionTrigger{}, nil
}
