// Package igapi is a thin, stateless client for the Instagram API with
// Instagram Login (graph.instagram.com, CLAUDE.md §4.0). It is transport-only:
// no Kit logic, no keep-code awareness, no safety enforcement, no DB, no
// apps/* imports. All outbound calls must pass through the safety layer
// before reaching this package.
package igapi

import "fmt"

// GraphErrorResponse is the error envelope returned by the Instagram API
// (RESOLVED G5 — identical shape to the legacy Graph API error envelope).
type GraphErrorResponse struct {
	Error GraphError `json:"error"`
}

// GraphError carries the structured error detail from the Instagram API.
type GraphError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	FBTraceID string `json:"fbtrace_id"`
}

// Error implements the error interface so GraphError can be wrapped with %w.
func (e GraphError) Error() string {
	return fmt.Sprintf("%s (code %d)", e.Message, e.Code)
}

// replyRequest is the POST body for /{comment-id}/replies.
type replyRequest struct {
	Message string `json:"message"`
}

// replyResponse is the success response from /{comment-id}/replies.
type replyResponse struct {
	ID string `json:"id"`
}

// messagesRequest is the POST body for /{ig-user-id}/messages.
// Used for both private replies (recipient.comment_id) and direct DMs (recipient.id).
type messagesRequest struct {
	Recipient messagesRecipient `json:"recipient"`
	Message   messagesMessage   `json:"message"`
}

type messagesRecipient struct {
	// CommentID is set when sending a private reply anchored to a comment.
	CommentID string `json:"comment_id,omitempty"`
	// ID is set when sending a free-form DM to a user.
	ID string `json:"id,omitempty"`
}

type messagesMessage struct {
	Text string `json:"text"`
}

// messagesResponse is the success response from /{ig-user-id}/messages.
type messagesResponse struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
}
