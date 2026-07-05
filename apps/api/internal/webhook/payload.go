package webhook

import "encoding/json"

// MetaPayload is the top-level envelope for Meta (Instagram) webhook notifications.
// object identifies the subscription product (e.g. "instagram").
type MetaPayload struct {
	Object string      `json:"object"`
	Entry  []MetaEntry `json:"entry"`
}

// MetaEntry is one subscription entry, typically one per connected page/account.
type MetaEntry struct {
	ID      string       `json:"id"`   // Instagram Business Account user ID
	Time    int64        `json:"time"` // Unix timestamp
	Changes []MetaChange `json:"changes"`
}

// MetaChange is a single field-level change notification inside an entry.
// Value is kept as raw JSON because its shape varies by field type.
type MetaChange struct {
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"` // parsed into CommentValue when field == "comments"
}

// CommentValue is the parsed value for changes where Field == "comments".
// Spec reference: ADR-001 §3.2 step 2; tolerant id parsing per ADR-002 §6.3 (RESOLVED G10).
type CommentValue struct {
	// ID is the Instagram comment ID (ig_comment_id in the DB).
	ID string `json:"id"`
	// CommentID is an alternate field name Meta's "examples" documentation
	// uses for the same value the "webhook reference" docs call `id`
	// (ADR-002 §6.3/G10 — genuinely ambiguous without a live payload).
	// ExtractComments falls back to this when ID is empty. Never populated
	// alongside a non-empty ID in normal operation.
	CommentID string `json:"comment_id"`
	// Text is the raw comment text.
	Text string `json:"text"`
	// From identifies the commenter.
	From CommentFrom `json:"from"`
	// Media is the post/Reel the comment was made on.
	// Guardrail §4b.5: this is always a post/Reel media ID, never an IG Live ID.
	Media CommentMedia `json:"media"`
	// ParentID is set for replies to other comments; typically absent for top-level comments.
	ParentID string `json:"parent_id,omitempty"`
}

// CommentFrom holds the identity of the comment author.
type CommentFrom struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// CommentMedia is the media object the comment belongs to.
type CommentMedia struct {
	ID string `json:"id"` // IG media ID (ig_media_id in catalog_post)
}

// IngestComment is the normalised comment datum produced by ExtractComments.
// EntryID is the parent entry's ID, which approximates the account's IG user ID
// in MVP single-account mode.
type IngestComment struct {
	EntryID string
	Value   CommentValue
}

// ExtractComments scans p and returns all successfully parsed comment-field
// values across all entries. Unknown field types and malformed JSON values are
// silently skipped — Meta will retry on non-200 responses so individual
// parse failures should not block the acknowledgement.
//
// Guardrail §4b.5: only the "comments" field is processed. This function
// never references IG Live events, follower changes, or any DO-NOT list item.
func ExtractComments(p MetaPayload) []IngestComment {
	var out []IngestComment
	for _, entry := range p.Entry {
		for _, ch := range entry.Changes {
			if ch.Field != "comments" {
				continue
			}
			var cv CommentValue
			if err := json.Unmarshal(ch.Value, &cv); err != nil {
				continue // skip malformed; do not fail the whole request
			}
			// Tolerant parser (ADR-002 §6.3/G10): some Meta docs use
			// `comment_id` instead of `id` for the same value. Don't guess
			// blindly — accept either, preferring `id` when both are set.
			if cv.ID == "" {
				cv.ID = cv.CommentID
			}
			out = append(out, IngestComment{EntryID: entry.ID, Value: cv})
		}
	}
	return out
}
