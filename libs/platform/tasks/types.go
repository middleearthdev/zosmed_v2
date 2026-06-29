// Package tasks defines task name constants and payload types shared between
// the API server (enqueue side) and the worker (handler side).
// This is the single source of truth — both apps import this package.
package tasks

// Task name constants. These strings are used as asynq task type identifiers.
const (
	// TaskCommentIngest is enqueued when a webhook comment event passes
	// initial verification and dedupe. The worker runs the workflow engine.
	TaskCommentIngest = "comment:ingest"

	// TaskReservationExpire is enqueued (with a delay equal to hold_seconds)
	// when a reservation is created. The worker transitions the reservation
	// to expired-released if it is still in a non-terminal state.
	TaskReservationExpire = "reservation:expire"
)

// CommentIngestPayload is the payload for TaskCommentIngest.
// All fields are strings to match Instagram Graph API identifiers.
type CommentIngestPayload struct {
	AccountID    string `json:"account_id"`
	CommentID    string `json:"comment_id"`
	MediaID      string `json:"media_id"`
	FromID       string `json:"from_id"`
	FromUsername string `json:"from_username"`
	Text         string `json:"text"`
}

// ReservationExpirePayload is the payload for TaskReservationExpire.
type ReservationExpirePayload struct {
	ReservationID string `json:"reservation_id"`
}
