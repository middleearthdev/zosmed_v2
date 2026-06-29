package igapi

import (
	"context"
	"fmt"
)

// ReplyToComment posts a public reply to an existing comment on a post or Reel.
//
// Endpoint: POST /{comment-id}/replies
//
// Rate limits (enforced by safety layer, not here):
//   - 750 comment replies per hour (Meta hard cap).
//   - 30 comments per post per 5 minutes (human-paced cap, §4c).
func (c *Client) ReplyToComment(ctx context.Context, commentID, text string) error {
	if commentID == "" {
		return fmt.Errorf("igapi: ReplyToComment: commentID is required")
	}
	if text == "" {
		return fmt.Errorf("igapi: ReplyToComment: text is required")
	}

	payload := replyRequest{Message: text}
	var result replyResponse
	if err := c.post(ctx, "/"+commentID+"/replies", payload, &result); err != nil {
		return fmt.Errorf("igapi: ReplyToComment %s: %w", commentID, err)
	}
	return nil
}
