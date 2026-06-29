// Package workflow is the segment-neutral workflow engine for Zosmed.
// It defines the Event type, node interfaces, a Registry, and an Engine
// that evaluates trigger→filter→action pipelines.
//
// No Kit-specific logic (keep codes, reservations, WhatsApp templates) lives
// here. Kits register nodes via Registry.Register* and the engine calls them.
package workflow

// Source constants for Event.Source.
const (
	SourceComment = "comment"
	SourceDM      = "dm"
	SourceStory   = "story"
)

// Event is a source-agnostic representation of a single Instagram interaction.
// It carries only the fields the engine and generic filter nodes need.
// Kit nodes access kit-specific semantics through the text/raw fields.
type Event struct {
	// Source is the interaction type: SourceComment, SourceDM, or SourceStory.
	Source string

	// AccountID is the Zosmed internal identifier of the IG Business account.
	AccountID string

	// ObjectID is the primary IG identifier for the triggering object:
	//   - comment ID for comment events
	//   - message ID for DM / story reply events
	ObjectID string

	// MediaID is the IG post/Reel/Story on which the interaction occurred.
	// Empty for DM-initiated events.
	MediaID string

	// FromID is the IG user ID of the person who triggered the event.
	FromID string

	// FromUsername is the IG handle (without @) of that person.
	FromUsername string

	// Text is the comment text, message body, or story reply text.
	Text string

	// Raw holds the full decoded webhook payload so Kit nodes can access
	// fields beyond the standard set (e.g., parent_id for threaded replies).
	Raw map[string]any
}
