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

	// TaskTokenRefreshSweep is enqueued periodically (asynq Scheduler, ADR-002
	// §5) to refresh IG-user long-lived tokens that are approaching expiry.
	// Carries no payload — the handler reads ListAccountsDueForRefresh fresh
	// from Postgres on every run.
	TaskTokenRefreshSweep = "token:refresh-sweep"

	// TaskReservationReconcile is enqueued periodically (asynq Scheduler) as a
	// backstop for lost reservation:expire timers (MAJOR-3b): it releases any
	// active reservation already past expires_at. Carries no payload — the
	// handler reads ListExpiredActiveReservations fresh from Postgres.
	TaskReservationReconcile = "reservation:reconcile"

	// TaskOutboundSend retries a private-reply that the safety gate deferred
	// (Queue) when quota was exhausted (MAJOR-3b/§4c overflow → queue → send when
	// quota recovers). Enqueued with a delay; the handler re-checks the gate.
	TaskOutboundSend = "outbound:send"
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

// OutboundSendPayload is the payload for TaskOutboundSend (MAJOR-2). It carries
// everything needed to re-attempt the private reply, EXCEPT the token — the
// worker looks that up per-account from Postgres (ADR-002 §6.2). CommentAt is
// RFC3339 for the gate's 7-day window re-check.
type OutboundSendPayload struct {
	AccountID     string `json:"account_id"`
	IgUserID      string `json:"ig_user_id"`
	CommentID     string `json:"comment_id"`
	TargetUserID  string `json:"target_user_id"`
	ReservationID string `json:"reservation_id"`
	ReplyText     string `json:"reply_text"`
	PostID        string `json:"post_id"`
	TriggerKey    string `json:"trigger_key"`
	CommentAt     string `json:"comment_at"`
}
