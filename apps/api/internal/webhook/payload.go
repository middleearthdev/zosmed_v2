package webhook

import (
	"encoding/json"
	"fmt"
)

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
	// Messaging carries DM / story-reply / story-mention / ad-referral events
	// (ADR-006 §3.1 — subscribed via the `messages` field, product Instagram).
	// This is a SEPARATE surface from Changes/comments and is never populated
	// by a comments notification.
	Messaging []MetaMessaging `json:"messaging"`
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

// ── Messaging surface (ADR-006 §3.1) — DM / story-reply / story-mention /
// ad-referral. All identity here is IGSID (§4.0): MetaMessagingUser.ID is
// entry.id for the account side, sender.id for the contact side — never a
// Facebook Page id.

// MetaMessagingUser identifies either side of a messaging event.
type MetaMessagingUser struct {
	ID string `json:"id"`
}

// MetaMessaging is one entry in entry[].messaging[] (webhook field `messages`,
// product Instagram). Every DM, story reply, story mention, and ad-referral
// event Zosmed cares about arrives in this shape (ADR-006 koreksi B0).
type MetaMessaging struct {
	Sender    MetaMessagingUser `json:"sender"`
	Recipient MetaMessagingUser `json:"recipient"`
	// Timestamp is the ms-epoch time of the interaction — the source for
	// conversation.last_interaction_at (§4c window store).
	Timestamp int64        `json:"timestamp"`
	Message   *MetaMessage `json:"message,omitempty"`
	// Referral is the TOP-LEVEL ad-referral (`messaging_referral`, thread
	// LAMA — koreksi B0). Distinct from Message.Referral (thread BARU).
	Referral *MetaReferral `json:"referral,omitempty"`
	// Postback.Referral is a Messenger/Facebook construct — NOT Instagram
	// Login (§4.0). Parsed only as an optional/tolerant fallback; MUST NOT be
	// the primary ad-referral path (ADR-006 §3.1/§10 Alternatif Ditolak).
	Postback *MetaPostback `json:"postback,omitempty"`
}

// MetaMessage is the `message` object of a messaging event.
type MetaMessage struct {
	Mid  string `json:"mid"`
	Text string `json:"text"`
	// ReplyTo.Story is set for a story-reply event.
	ReplyTo *MetaReplyTo `json:"reply_to,omitempty"`
	// Attachments carries the story-mention marker: an attachment with
	// Type=="story_mention" (ADR-006 koreksi B0 — NOT changes[].mentions).
	Attachments []MetaMessageAttachment `json:"attachments,omitempty"`
	// Referral is the ad-referral payload for a THREAD-NEW (first message)
	// click-to-DM conversation.
	Referral *MetaReferral `json:"referral,omitempty"`
}

// MetaReplyTo wraps the story being replied to.
type MetaReplyTo struct {
	Story *MetaStoryRef `json:"story,omitempty"`
}

// MetaStoryRef identifies the story a message replies to (story-reply subtype).
type MetaStoryRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// MetaMessageAttachment is one attachment on a message. Type=="story_mention"
// is the ONLY marker Zosmed uses to classify the story-mention subtype.
type MetaMessageAttachment struct {
	Type    string                       `json:"type"`
	Payload MetaMessageAttachmentPayload `json:"payload"`
}

// MetaMessageAttachmentPayload holds the attachment's URL (unused in MVP;
// kept for parity with the documented payload shape, ADR-006 §3.1).
type MetaMessageAttachmentPayload struct {
	URL string `json:"url"`
}

// MetaReferral is the ad-referral payload, whether nested under Message
// (thread-new) or top-level on MetaMessaging (thread-old, messaging_referral).
type MetaReferral struct {
	Ref    string `json:"ref"`
	AdID   string `json:"ad_id"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

// MetaPostback is a Messenger/Facebook-only construct, tolerated but never
// relied upon (§4.0 — Instagram Login has no postback surface).
type MetaPostback struct {
	Referral *MetaPostbackReferral `json:"referral,omitempty"`
}

// MetaPostbackReferral mirrors MetaReferral's shape for the tolerant fallback.
type MetaPostbackReferral struct {
	Ref  string `json:"ref"`
	AdID string `json:"ad_id"`
}

// Messaging event subtype constants (ADR-006 §2 — mirrored as raw-key literal
// values in libs/workflow/nodes; kept as string literals here too, per the
// same "shared wire-key convention, not shared logic" rule as action_wa_link.go,
// so the webhook transport package never imports the engine's node package).
const (
	MessagingSubtypeDM           = "dm"
	MessagingSubtypeStoryReply   = "story-reply"
	MessagingSubtypeStoryMention = "story-mention"
	MessagingSubtypeAdReferral   = "ad-referral"

	// storyMentionAttachmentType is the only attachment Type that classifies a
	// messaging event as story-mention (ADR-006 koreksi B0).
	storyMentionAttachmentType = "story_mention"
)

// IngestMessaging is the normalised messaging datum produced by
// ExtractMessagingEvents, subtype already classified (ADR-006 §3.2).
type IngestMessaging struct {
	EntryID   string
	EntryTime int64
	ContactID string // IGSID of the sender (§4.0)
	MessageID string // message.mid — dedupe key for processed_message
	Text      string
	// Subtype ∈ MessagingSubtype* constants above.
	Subtype string
	// Source is always "dm" for every messaging-surface event (ADR-006
	// koreksi B0 point 4) — kept as a field (not hardcoded at the call site)
	// so callers don't need to know the constant lives here.
	Source string
	// AdRef is the ad-referral ref, populated only when Subtype == ad-referral.
	AdRef string
	// EventAt is the ms-epoch webhook timestamp (Message/story window source).
	EventAt int64
}

// IngestComment is the normalised comment datum produced by ExtractComments.
// EntryID is the parent entry's ID, which approximates the account's IG user ID
// in MVP single-account mode.
type IngestComment struct {
	EntryID string
	// EntryTime is the webhook entry timestamp (Unix seconds). It is the
	// notification time, which for a fresh comment closely approximates the
	// comment's created time — used for the §4c 7-day private-reply window (M4).
	// Full accuracy would require GET /{comment-id}?fields=timestamp (ADR-002 §6.3).
	EntryTime int64
	Value     CommentValue
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
			out = append(out, IngestComment{EntryID: entry.ID, EntryTime: entry.Time, Value: cv})
		}
	}
	return out
}

// ExtractMessagingEvents scans p and returns every entry[].messaging[] event
// (webhook field `messages`), subtype-classified per ADR-006 §3.2. This is a
// SEPARATE surface from ExtractComments/changes[] — the two never overlap.
//
// Guardrail (ADR-006 koreksi B0 / §9): story-mention is classified from
// message.attachments[].type=="story_mention", NEVER from changes[].mentions
// (a different, comment/caption-level capability, out of scope here). This
// function does not read entry.Changes at all.
//
// Classification order matters (checked before plain "dm"): story-mention →
// story-reply → ad-referral → else plain dm. All four subtypes carry
// Source=="dm" (ADR-006 koreksi B0 point 4) — SourceStory is never produced.
func ExtractMessagingEvents(p MetaPayload) []IngestMessaging {
	var out []IngestMessaging
	for _, entry := range p.Entry {
		for _, m := range entry.Messaging {
			im := IngestMessaging{
				EntryID:   entry.ID,
				EntryTime: entry.Time,
				ContactID: m.Sender.ID,
				Subtype:   MessagingSubtypeDM,
				Source:    MessagingSubtypeDM,
				EventAt:   m.Timestamp,
			}

			if m.Message != nil {
				im.MessageID = m.Message.Mid
				im.Text = m.Message.Text

				switch {
				case hasStoryMentionAttachment(m.Message.Attachments):
					im.Subtype = MessagingSubtypeStoryMention
				case m.Message.ReplyTo != nil && m.Message.ReplyTo.Story != nil:
					im.Subtype = MessagingSubtypeStoryReply
				case m.Message.Referral != nil || m.Referral != nil:
					im.Subtype = MessagingSubtypeAdReferral
					im.AdRef = firstNonEmpty(referralRef(m.Message.Referral), referralRef(m.Referral))
				}
			} else if m.Referral != nil {
				// Defensive: a top-level referral without a message body
				// (older messaging_referral shape) still counts as ad-referral.
				im.Subtype = MessagingSubtypeAdReferral
				im.AdRef = referralRef(m.Referral)
			}
			// m.Postback.Referral is intentionally NEVER consulted here — it is
			// a Messenger/Facebook-only construct (§4.0), not a primary path.

			// An ad-referral event may carry no message body (older
			// messaging_referral shape) or an empty mid, leaving MessageID blank —
			// which processMessaging's empty-MessageID skip would silently drop,
			// so the click-to-dm-ad trigger would never fire for a returning
			// contact. Synthesize a STABLE dedupe key (entry + timestamp + ref) so
			// a redelivered referral still dedupes via processed_message.
			if im.MessageID == "" && im.Subtype == MessagingSubtypeAdReferral {
				im.MessageID = fmt.Sprintf("ref:%s:%d:%s", entry.ID, m.Timestamp, im.AdRef)
			}

			out = append(out, im)
		}
	}
	return out
}

// hasStoryMentionAttachment reports whether any attachment marks this message
// as a story mention (ADR-006 koreksi B0).
func hasStoryMentionAttachment(atts []MetaMessageAttachment) bool {
	for _, a := range atts {
		if a.Type == storyMentionAttachmentType {
			return true
		}
	}
	return false
}

// referralRef safely reads Ref from a possibly-nil MetaReferral.
func referralRef(r *MetaReferral) string {
	if r == nil {
		return ""
	}
	return r.Ref
}

// firstNonEmpty returns the first non-empty string among vals.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
