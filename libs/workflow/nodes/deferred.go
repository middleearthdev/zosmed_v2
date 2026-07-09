package nodes

import (
	"context"
	"time"
)

// ── §4c window constants (duplicated as literals, NOT imported from
// libs/safety — this package must stay free of a libs/safety dependency, see
// the package doc + the sendDMKind/replyCommentKind comments in the sibling
// action_*.go files for the same rationale) ─────────────────────────────────

const (
	// PrivateReplyWindow mirrors safety.PrivateReplyWindowDays (§4c: a private
	// reply must be sent within 7 days of the comment).
	PrivateReplyWindow = 7 * 24 * time.Hour
	// MessagingWindow mirrors safety.MessagingWindowHours (§4c: a standard DM
	// is only allowed within 24h of the user's last interaction).
	MessagingWindow = 24 * time.Hour
	// DeferredCommentReplyTTL is the Deadline TTL for a deferred comment-reply
	// retry (ADR-007 §2.1 point 2). Unlike private-reply/dm, a public
	// comment-reply has no hard §4c window in libs/safety/window.go — its
	// quota (750/hr, 30/post/5min) recovers within minutes to hours, so a
	// fixed 6h cutoff is a reasonable belt-and-suspenders stop that keeps a
	// stale retry chain from running forever (ADR-007 §2.1 alternative
	// "without Deadline" was rejected for exactly this reason).
	DeferredCommentReplyTTL = 6 * time.Hour
)

// DeferredRetryDelay is how long a deferred outbound waits before
// outbound:send re-checks the safety gate (ADR-007 §2.1). One shared
// constant for every Kind and every caller (neutral nodes + seller kit) —
// mirrors the pre-ADR-007 seller.OutboundRetryDelay value.
const DeferredRetryDelay = time.Minute

// DeferredOutbound carries everything the generic outbound:send task needs
// to re-attempt an outbound IG message that the safety gate deferred
// (workflow.DecisionQueue). It is the segment-neutral contract between a
// node's Execute (this package, or libs/kits/*) and the asynq enqueue
// closure injected by apps/worker/internal/runner (ADR-007 §2.1 point 1).
type DeferredOutbound struct {
	AccountID    string
	Kind         string // "private-reply" | "dm" | "comment-reply"
	IgUserID     string // IGSID of the business account (sender)
	TargetUserID string // IGSID of the recipient
	ObjectID     string // comment_id (reply/private-reply) or message id (dm) — anchor + dedupe trigger
	Text         string // final rendered text
	CommentAt    time.Time
	PostID       string
	TriggerKey   string
	Deadline     time.Time // TTL: drop if now > Deadline at dequeue (§4c)

	// ReservationID is OPTIONAL — only the seller kit's private-reply path
	// populates it (ADR-007 §2.1 point 4). Every neutral node leaves it empty.
	ReservationID string
}

// EnqueueDeferredFunc schedules a DeferredOutbound retry after delay.
// Injected into RegisterFactories (this package) and seller.RegisterFactories
// (ADR-007 §3.8); nil disables retry — the Queue decision is simply reported
// as deferred (fallback report-only, preserves pre-ADR-007 behaviour for
// callers/tests that have not wired the asynq closure).
type EnqueueDeferredFunc func(ctx context.Context, d DeferredOutbound, delay time.Duration) error

// DeadlineFrom returns commentAt+ttl, using time.Now() as the base when
// commentAt is zero (unknown/unavailable timestamp) so a Deadline TTL never
// begins in the deep past — mirrors safety/window.go's treatment of a zero
// CommentAt ("cannot enforce; allow") without letting a freshly-deferred
// retry drop on its very first dequeue attempt.
func DeadlineFrom(commentAt time.Time, ttl time.Duration) time.Time {
	base := commentAt
	if base.IsZero() {
		base = time.Now()
	}
	return base.Add(ttl)
}
