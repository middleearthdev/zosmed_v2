package webhook

import "testing"

// ── ExtractMessagingEvents (ADR-006 §3.2) ────────────────────────────────────

func TestExtractMessagingEvents_PlainDM(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID:   "igsid-account-1",
			Time: 1719600000,
			Messaging: []MetaMessaging{{
				Sender:    MetaMessagingUser{ID: "igsid-user-1"},
				Recipient: MetaMessagingUser{ID: "igsid-account-1"},
				Timestamp: 1719600000123,
				Message:   &MetaMessage{Mid: "mid-1", Text: "halo kak"},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	got := events[0]
	if got.Subtype != MessagingSubtypeDM {
		t.Errorf("Subtype = %q, want %q", got.Subtype, MessagingSubtypeDM)
	}
	if got.Source != MessagingSubtypeDM {
		t.Errorf("Source = %q, want %q (ADR-006 koreksi B0 point 4: always dm)", got.Source, MessagingSubtypeDM)
	}
	if got.ContactID != "igsid-user-1" {
		t.Errorf("ContactID = %q, want %q", got.ContactID, "igsid-user-1")
	}
	if got.MessageID != "mid-1" {
		t.Errorf("MessageID = %q, want %q", got.MessageID, "mid-1")
	}
	if got.Text != "halo kak" {
		t.Errorf("Text = %q, want %q", got.Text, "halo kak")
	}
	if got.EntryID != "igsid-account-1" {
		t.Errorf("EntryID = %q, want %q", got.EntryID, "igsid-account-1")
	}
	if got.EventAt != 1719600000123 {
		t.Errorf("EventAt = %d, want %d", got.EventAt, int64(1719600000123))
	}
	if got.AdRef != "" {
		t.Errorf("AdRef = %q, want empty for plain DM", got.AdRef)
	}
}

func TestExtractMessagingEvents_StoryReply(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "igsid-account-1",
			Messaging: []MetaMessaging{{
				Sender: MetaMessagingUser{ID: "igsid-user-2"},
				Message: &MetaMessage{
					Mid:  "mid-2",
					Text: "keren banget storynya kak",
					ReplyTo: &MetaReplyTo{
						Story: &MetaStoryRef{ID: "story-1", URL: "https://example.com/story.jpg"},
					},
				},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Subtype != MessagingSubtypeStoryReply {
		t.Errorf("Subtype = %q, want %q", events[0].Subtype, MessagingSubtypeStoryReply)
	}
	if events[0].Source != MessagingSubtypeDM {
		t.Errorf("Source = %q, want %q (story-reply is Source=dm, not SourceStory)", events[0].Source, MessagingSubtypeDM)
	}
}

// TestExtractMessagingEvents_StoryMentionViaAttachment is the CRITICAL
// koreksi-B0 regression guard: story-mention MUST be classified from
// message.attachments[].type=="story_mention" — never from changes[].mentions
// (a different capability, out of scope per ADR-006 §0/§9).
func TestExtractMessagingEvents_StoryMentionViaAttachment(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "igsid-account-1",
			Messaging: []MetaMessaging{{
				Sender: MetaMessagingUser{ID: "igsid-user-3"},
				Message: &MetaMessage{
					Mid: "mid-3",
					Attachments: []MetaMessageAttachment{
						{Type: "story_mention", Payload: MetaMessageAttachmentPayload{URL: "https://example.com/mention.jpg"}},
					},
				},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Subtype != MessagingSubtypeStoryMention {
		t.Errorf("Subtype = %q, want %q", events[0].Subtype, MessagingSubtypeStoryMention)
	}
	if events[0].Source != MessagingSubtypeDM {
		t.Errorf("Source = %q, want %q (story-mention is Source=dm — opens window like story-reply, R3)", events[0].Source, MessagingSubtypeDM)
	}
	if events[0].MessageID != "mid-3" {
		t.Errorf("MessageID = %q, want %q (story-mention carries mid — dedupe key)", events[0].MessageID, "mid-3")
	}
}

func TestExtractMessagingEvents_StoryMentionPriorityOverStoryReply(t *testing.T) {
	// Classification order (ADR-006 §3.2): story-mention checked BEFORE
	// story-reply. A payload carrying both markers must classify as mention.
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{{
				Sender: MetaMessagingUser{ID: "u1"},
				Message: &MetaMessage{
					Mid:         "mid-4",
					ReplyTo:     &MetaReplyTo{Story: &MetaStoryRef{ID: "story-x"}},
					Attachments: []MetaMessageAttachment{{Type: "story_mention"}},
				},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Subtype != MessagingSubtypeStoryMention {
		t.Errorf("Subtype = %q, want %q (mention takes priority)", events[0].Subtype, MessagingSubtypeStoryMention)
	}
}

func TestExtractMessagingEvents_AdReferral_MessageLevel(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{{
				Sender: MetaMessagingUser{ID: "u1"},
				Message: &MetaMessage{
					Mid:      "mid-5",
					Text:     "hi, saw your ad",
					Referral: &MetaReferral{Ref: "campaign-a", AdID: "ad-1", Source: "ADS", Type: "OPEN_THREAD"},
				},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Subtype != MessagingSubtypeAdReferral {
		t.Errorf("Subtype = %q, want %q", events[0].Subtype, MessagingSubtypeAdReferral)
	}
	if events[0].AdRef != "campaign-a" {
		t.Errorf("AdRef = %q, want %q (from message.referral)", events[0].AdRef, "campaign-a")
	}
}

func TestExtractMessagingEvents_AdReferral_TopLevel(t *testing.T) {
	// Thread-OLD ad-referral (messaging_referral): top-level Referral, no
	// message.referral (ADR-006 koreksi B0 point 6).
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{{
				Sender:   MetaMessagingUser{ID: "u1"},
				Message:  &MetaMessage{Mid: "mid-6", Text: "lanjut chat"},
				Referral: &MetaReferral{Ref: "campaign-b"},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Subtype != MessagingSubtypeAdReferral {
		t.Errorf("Subtype = %q, want %q", events[0].Subtype, MessagingSubtypeAdReferral)
	}
	if events[0].AdRef != "campaign-b" {
		t.Errorf("AdRef = %q, want %q (from top-level referral)", events[0].AdRef, "campaign-b")
	}
}

// TestExtractMessagingEvents_AdReferral_NoMessageBody_SynthesizesDedupeKey is a
// regression for the review finding that a top-level messaging_referral with NO
// message body (no mid) was silently dropped downstream by processMessaging's
// empty-MessageID skip. A stable synthetic MessageID must be produced so the
// click-to-dm-ad trigger fires and redeliveries still dedupe.
func TestExtractMessagingEvents_AdReferral_NoMessageBody_SynthesizesDedupeKey(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{{
				Sender:    MetaMessagingUser{ID: "u1"},
				Timestamp: 1719600000999,
				Referral:  &MetaReferral{Ref: "campaign-c"},
				// Message deliberately nil (older messaging_referral shape).
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	got := events[0]
	if got.Subtype != MessagingSubtypeAdReferral {
		t.Errorf("Subtype = %q, want %q", got.Subtype, MessagingSubtypeAdReferral)
	}
	if got.MessageID == "" {
		t.Fatal("MessageID must be synthesized (non-empty) so the event is not dropped by the empty-MessageID skip")
	}
	// Stable across redelivery (same entry+timestamp+ref) → dedupes correctly.
	if again := ExtractMessagingEvents(payload); again[0].MessageID != got.MessageID {
		t.Errorf("synthetic MessageID not stable: %q vs %q", again[0].MessageID, got.MessageID)
	}
}

func TestExtractMessagingEvents_AdReferral_MessageLevelPreferredOverTopLevel(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{{
				Sender:   MetaMessagingUser{ID: "u1"},
				Message:  &MetaMessage{Mid: "mid-7", Referral: &MetaReferral{Ref: "message-level-wins"}},
				Referral: &MetaReferral{Ref: "top-level-loses"},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].AdRef != "message-level-wins" {
		t.Errorf("AdRef = %q, want %q (message.referral takes priority)", events[0].AdRef, "message-level-wins")
	}
}

func TestExtractMessagingEvents_PostbackReferralIgnored(t *testing.T) {
	// §4.0/§10 Alternatif Ditolak: postback.referral (Messenger/Facebook-only)
	// MUST NEVER be the primary ad-referral path.
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{{
				Sender:   MetaMessagingUser{ID: "u1"},
				Message:  &MetaMessage{Mid: "mid-8", Text: "hi"},
				Postback: &MetaPostback{Referral: &MetaPostbackReferral{Ref: "should-be-ignored"}},
			}},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Subtype != MessagingSubtypeDM {
		t.Errorf("Subtype = %q, want %q (postback.referral must not classify as ad-referral)", events[0].Subtype, MessagingSubtypeDM)
	}
	if events[0].AdRef != "" {
		t.Errorf("AdRef = %q, want empty (postback.referral must never populate AdRef)", events[0].AdRef)
	}
}

func TestExtractMessagingEvents_EmptyPayload(t *testing.T) {
	events := ExtractMessagingEvents(MetaPayload{})
	if len(events) != 0 {
		t.Fatalf("expected 0 events for empty payload, got %d", len(events))
	}
}

func TestExtractMessagingEvents_MultipleEventsInOneEntry(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Messaging: []MetaMessaging{
				{Sender: MetaMessagingUser{ID: "u1"}, Message: &MetaMessage{Mid: "m1", Text: "hi"}},
				{Sender: MetaMessagingUser{ID: "u2"}, Message: &MetaMessage{Mid: "m2", Text: "hello"}},
			},
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

// TestRegression_ExtractMessagingEvents_NeverReadsChanges guards the koreksi
// B0 rule: this function must not derive events from entry.Changes at all,
// regardless of what "mentions" field content is present there.
func TestRegression_ExtractMessagingEvents_NeverReadsChanges(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{{
			ID: "e1",
			Changes: []MetaChange{
				{Field: "mentions", Value: []byte(`{"media_id":"m1","comment_id":"c1"}`)},
			},
			// No Messaging entries at all.
		}},
	}

	events := ExtractMessagingEvents(payload)
	if len(events) != 0 {
		t.Fatalf("expected 0 messaging events when only changes[].mentions is present, got %d", len(events))
	}
}
