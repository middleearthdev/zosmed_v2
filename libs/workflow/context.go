package workflow

import "context"

// Sender is the consumer-defined interface for outbound IG messaging.
// The concrete implementation wraps igapi.Client inside the safety layer so
// every outbound call is gate-checked before touching the Graph API.
// Engine and Kit nodes use this interface; neither imports igapi or safety directly.
type Sender interface {
	// ReplyToComment posts a public reply to a comment on a post/Reel.
	ReplyToComment(ctx context.Context, commentID, text string) error
	// SendPrivateReply sends a DM anchored to a specific comment (1 per comment, ≤7 days).
	SendPrivateReply(ctx context.Context, igUserID, commentID, text string) error
	// SendDM sends a free-form DM within the 24-hour messaging window.
	SendDM(ctx context.Context, igUserID, targetUserID, text string) error
}

// StepStatus indicates the outcome of one node evaluation.
type StepStatus string

const (
	StepOK      StepStatus = "ok"
	StepSkipped StepStatus = "skipped"
	StepError   StepStatus = "error"
)

// StepLog records the outcome of one node in the run.
type StepLog struct {
	NodeKey string
	Kind    NodeKind
	Status  StepStatus
	// Detail is a human-readable note (reason for skip/error, or action summary).
	Detail string
}

// RunContext carries the triggering Event, accumulated variables, and injected
// services for one engine run. It is not safe for concurrent use.
type RunContext struct {
	// Event is the triggering IG interaction.
	Event Event

	// Vars holds template variables accumulated by nodes during the run.
	// Examples: "nama" → username, "produk" → product name, "kode" → keep code.
	// Nodes read and write Vars to share state without tight coupling.
	Vars map[string]string

	// Sender is the outbound messaging service.
	// Wired by the runner so engine/Kit nodes never call igapi directly.
	Sender Sender

	// Gate is the safety gate.
	// Wired by the runner; Kit nodes call Gate.Allow before sending.
	Gate Gater

	// Steps is the ordered log accumulated during this run.
	Steps []StepLog
}

// NewRunContext returns a RunContext initialised for event with the given services.
func NewRunContext(event Event, sender Sender, gate Gater) *RunContext {
	return &RunContext{
		Event:  event,
		Vars:   make(map[string]string),
		Sender: sender,
		Gate:   gate,
	}
}

// AddStep appends a step log entry.
func (rc *RunContext) AddStep(s StepLog) {
	rc.Steps = append(rc.Steps, s)
}
