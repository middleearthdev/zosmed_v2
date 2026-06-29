package igapi

import (
	"context"
	"fmt"
)

// SendPrivateReply sends a DM anchored to a specific comment (private reply).
// This opens — or continues — the messaging thread with the comment author.
//
// Endpoint: POST /{ig-user-id}/messages  (recipient = {comment_id: "..."})
//
// Constraints (enforced by the safety layer before this is called):
//   - Maximum 1 private reply per comment.
//   - Must be sent within 7 days of the comment (PRIVATE_REPLY_WINDOW_DAYS).
//   - Counts against the DM rate limits (200/hr, 1000/day).
//
// igUserID is the Zosmed account's IG user ID (the sender, i.e. the business page).
func (c *Client) SendPrivateReply(ctx context.Context, igUserID, commentID, text string) error {
	if igUserID == "" {
		return fmt.Errorf("igapi: SendPrivateReply: igUserID is required")
	}
	if commentID == "" {
		return fmt.Errorf("igapi: SendPrivateReply: commentID is required")
	}
	if text == "" {
		return fmt.Errorf("igapi: SendPrivateReply: text is required")
	}

	payload := messagesRequest{
		Recipient: messagesRecipient{CommentID: commentID},
		Message:   messagesMessage{Text: text},
	}
	var result messagesResponse
	if err := c.post(ctx, "/"+igUserID+"/messages", payload, &result); err != nil {
		return fmt.Errorf("igapi: SendPrivateReply (comment %s): %w", commentID, err)
	}
	return nil
}

// SendDM sends a free-form DM to an IG user who has an open messaging window.
//
// Endpoint: POST /{ig-user-id}/messages  (recipient = {id: "..."})
//
// Constraints (enforced by the safety layer before this is called):
//   - User must have interacted within the last 24 hours (MESSAGING_WINDOW_HOURS).
//   - Rate limits: 200 DM/hour (overflow → queue), 1000 DM/day.
//   - Messages to non-followers may land in message requests, not the main inbox.
//
// igUserID is the Zosmed account's IG user ID (the sender, i.e. the business page).
// targetUserID is the IG user ID of the recipient.
func (c *Client) SendDM(ctx context.Context, igUserID, targetUserID, text string) error {
	if igUserID == "" {
		return fmt.Errorf("igapi: SendDM: igUserID is required")
	}
	if targetUserID == "" {
		return fmt.Errorf("igapi: SendDM: targetUserID is required")
	}
	if text == "" {
		return fmt.Errorf("igapi: SendDM: text is required")
	}

	payload := messagesRequest{
		Recipient: messagesRecipient{ID: targetUserID},
		Message:   messagesMessage{Text: text},
	}
	var result messagesResponse
	if err := c.post(ctx, "/"+igUserID+"/messages", payload, &result); err != nil {
		return fmt.Errorf("igapi: SendDM (target %s): %w", targetUserID, err)
	}
	return nil
}
