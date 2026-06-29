package workflow

import (
	"context"
	"time"
)

// DecisionAction is the safety gate verdict enum.
type DecisionAction int

const (
	// DecisionAllow means the outbound action may proceed immediately.
	DecisionAllow DecisionAction = iota
	// DecisionQueue means the action should be deferred (overflow queue).
	DecisionQueue
	// DecisionReject means the action must not be sent (window expired, kill switch, etc.).
	DecisionReject
)

// OutboundReq describes a single outbound IG action to be gate-checked.
// The fields mirror libs/safety.OutboundReq so that the runner can build a
// thin adapter rather than a full conversion layer.
type OutboundReq struct {
	// AccountID is the Zosmed internal account identifier.
	AccountID string
	// Kind is the outbound type: "private-reply" or "dm".
	Kind string
	// TargetUserID is the IG user ID of the message recipient.
	TargetUserID string
	// TriggerKey is the dedupe key for the triggering event
	// (e.g., comment ID for private replies, message ID for DMs).
	TriggerKey string
	// CommentID is the IG comment ID (non-empty for private replies).
	CommentID string
	// CommentAt is when the triggering comment was posted, used to enforce
	// the 7-day private-reply window and the 24-hour DM window.
	CommentAt time.Time
	// PostID is optional. When non-empty, the safety layer also evaluates
	// the per-post-per-5-min counter (§4c: 30 comments/post/5 min).
	PostID string
}

// Decision is the gate's verdict on an OutboundReq.
type Decision struct {
	Action DecisionAction
	Reason string
}

// Gater is the consumer-defined interface for the safety gate.
// The concrete implementation in libs/safety must satisfy this interface.
// Engine and Kit nodes accept Gater to stay decoupled from the safety package.
type Gater interface {
	Allow(ctx context.Context, req OutboundReq) (Decision, error)
}
