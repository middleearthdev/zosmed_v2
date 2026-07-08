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

	// TaskDMIngest is enqueued when a webhook messaging event (DM, story
	// reply, story mention, or ad-referral — ADR-006 §3.3) passes account
	// resolution, dedupe, and the HasLiveWorkflow gate. Mirrors
	// TaskCommentIngest but runs a separate ingest path: no catalog_post
	// coupling (DM/story is not seller-specific), and it upserts the
	// conversation window store on every event (ADR-006 §4.1).
	TaskDMIngest = "dm:ingest"
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
	// CommentAt is the webhook entry timestamp in RFC3339 (M4). Captured at
	// ingest so the §4c 7-day window is measured from the comment time, not from
	// when the worker happens to dequeue the task. Empty on legacy payloads.
	CommentAt string `json:"comment_at"`
}

// ReservationExpirePayload is the payload for TaskReservationExpire.
type ReservationExpirePayload struct {
	ReservationID string `json:"reservation_id"`
}

// DMIngestPayload is the payload for TaskDMIngest (ADR-006 §4.1). AccountID is
// the only credential-adjacent field — the worker looks up the access token
// from Postgres itself (ADR-002 §6.2, never carried in the queue payload).
// Source is always "dm" for every messaging-surface event (DM, story-reply,
// story-mention, ad-referral — ADR-006 koreksi B0); Subtype is the single
// discriminator the six neutral trigger nodes match on.
type DMIngestPayload struct {
	AccountID string `json:"account_id"`
	// Source is always workflow.SourceDM ("dm") — kept as a string here (not
	// the workflow package's constant) so this platform-level package stays
	// free of a dependency on libs/workflow (§5a boundary).
	Source string `json:"source"`
	// Subtype ∈ {"dm","story-reply","story-mention","ad-referral"} (ADR-006 §2).
	Subtype      string `json:"subtype"`
	MessageID    string `json:"message_id"`
	MediaID      string `json:"media_id,omitempty"`
	FromID       string `json:"from_id"`
	FromUsername string `json:"from_username,omitempty"`
	Text         string `json:"text,omitempty"`
	// AdRef carries the ad-referral payload for the click-to-dm-ad trigger
	// (ADR-006 §2.1); empty for every other subtype.
	AdRef string `json:"ad_ref,omitempty"`
	// EventAt is the webhook messaging timestamp (RFC3339) — the source of
	// truth for conversation.last_interaction_at (ADR-006 §4.1 step 4).
	EventAt string `json:"event_at"`
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
