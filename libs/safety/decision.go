package safety

// Action is the Gate's verdict for an outbound request.
type Action int

const (
	// Allow means the request cleared all checks and may be sent immediately.
	Allow Action = iota
	// Queue means the request should be placed in the outbound queue (e.g. DM
	// overflow, auto-pause near limit). The message is NOT lost — it will be
	// retried when quota recovers. DM overflow is explicitly Queue, not Reject
	// (CLAUDE.md §4c / §10).
	Queue
	// Reject means the request must NOT be sent: window expired, dedupe hit,
	// or kill-switch engaged.
	Reject
)

// String returns a human-readable label for logging.
func (a Action) String() string {
	switch a {
	case Allow:
		return "allow"
	case Queue:
		return "queue"
	case Reject:
		return "reject"
	default:
		return "unknown"
	}
}

// Decision is the Gate's full verdict, including the reason for non-Allow outcomes.
type Decision struct {
	Action Action
	Reason string
}
