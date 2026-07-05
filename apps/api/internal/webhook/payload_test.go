package webhook

import (
	"encoding/json"
	"testing"
)

// commentValueJSON serialises a CommentValue to json.RawMessage for embedding in a change.
func commentValueJSON(t *testing.T, cv CommentValue) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(cv)
	if err != nil {
		t.Fatalf("marshal comment value: %v", err)
	}
	return b
}

// ── ExtractComments ───────────────────────────────────────────────────────────

func TestExtractComments_SingleComment(t *testing.T) {
	cv := CommentValue{
		ID:    "17858893269000001",
		Text:  "keep C1",
		From:  CommentFrom{ID: "111222333", Username: "buyer_satu"},
		Media: CommentMedia{ID: "17896129340000001"},
	}

	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{
			{
				ID:   "123456789",
				Time: 1719600000,
				Changes: []MetaChange{
					{Field: "comments", Value: commentValueJSON(t, cv)},
				},
			},
		},
	}

	comments := ExtractComments(payload)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	got := comments[0]
	if got.EntryID != "123456789" {
		t.Errorf("EntryID: want %q, got %q", "123456789", got.EntryID)
	}
	if got.Value.ID != cv.ID {
		t.Errorf("comment ID: want %q, got %q", cv.ID, got.Value.ID)
	}
	if got.Value.Text != cv.Text {
		t.Errorf("text: want %q, got %q", cv.Text, got.Value.Text)
	}
	if got.Value.From.Username != cv.From.Username {
		t.Errorf("username: want %q, got %q", cv.From.Username, got.Value.From.Username)
	}
	if got.Value.Media.ID != cv.Media.ID {
		t.Errorf("media ID: want %q, got %q", cv.Media.ID, got.Value.Media.ID)
	}
}

func TestExtractComments_MultipleEntries(t *testing.T) {
	mkChange := func(commentID, text string) MetaChange {
		cv := CommentValue{
			ID:    commentID,
			Text:  text,
			From:  CommentFrom{ID: "u1", Username: "u"},
			Media: CommentMedia{ID: "m1"},
		}
		return MetaChange{Field: "comments", Value: commentValueJSON(t, cv)}
	}

	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{
			{ID: "e1", Changes: []MetaChange{mkChange("c1", "keep"), mkChange("c2", "C1")}},
			{ID: "e2", Changes: []MetaChange{mkChange("c3", "order")}},
		},
	}

	comments := ExtractComments(payload)
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(comments))
	}
}

func TestExtractComments_SkipsNonCommentFields(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{
			{
				ID: "e1",
				Changes: []MetaChange{
					{Field: "messages", Value: json.RawMessage(`{"mid":"msg1"}`)},
					{Field: "mentions", Value: json.RawMessage(`{}`)},
				},
			},
		},
	}

	comments := ExtractComments(payload)
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments for non-comment fields, got %d", len(comments))
	}
}

func TestExtractComments_SkipsMalformedValue(t *testing.T) {
	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{
			{
				ID: "e1",
				Changes: []MetaChange{
					{Field: "comments", Value: json.RawMessage(`not-valid-json}`)},
				},
			},
		},
	}

	// Should not panic; malformed values are silently skipped.
	comments := ExtractComments(payload)
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments for malformed value, got %d", len(comments))
	}
}

func TestExtractComments_MixedValidAndMalformed(t *testing.T) {
	good := CommentValue{
		ID:    "c_good",
		Text:  "keep",
		From:  CommentFrom{ID: "u1", Username: "buyer"},
		Media: CommentMedia{ID: "m1"},
	}

	payload := MetaPayload{
		Object: "instagram",
		Entry: []MetaEntry{
			{
				ID: "e1",
				Changes: []MetaChange{
					{Field: "comments", Value: json.RawMessage(`{malformed`)},
					{Field: "comments", Value: commentValueJSON(t, good)},
				},
			},
		},
	}

	comments := ExtractComments(payload)
	if len(comments) != 1 {
		t.Fatalf("expected 1 valid comment, got %d", len(comments))
	}
	if comments[0].Value.ID != "c_good" {
		t.Errorf("expected valid comment ID, got %q", comments[0].Value.ID)
	}
}

func TestExtractComments_EmptyPayload(t *testing.T) {
	comments := ExtractComments(MetaPayload{})
	if len(comments) != 0 {
		t.Fatalf("expected nil/empty slice for empty payload, got %v", comments)
	}
}

// ── Tolerant id/comment_id parsing (ADR-002 §6.3, RESOLVED G10) ────────────────

func TestExtractComments_FallsBackToCommentIDWhenIDEmpty(t *testing.T) {
	// Some Meta docs use `comment_id` instead of `id` for the same value.
	raw := json.RawMessage(`{"comment_id":"17858893269000099","text":"C1","from":{"id":"u9","username":"buyer_nine"},"media":{"id":"m9"}}`)

	payload := MetaPayload{
		Object: "instagram",
		Entry:  []MetaEntry{{ID: "e1", Changes: []MetaChange{{Field: "comments", Value: raw}}}},
	}

	comments := ExtractComments(payload)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Value.ID != "17858893269000099" {
		t.Errorf("expected ID to fall back to comment_id, got %q", comments[0].Value.ID)
	}
}

func TestExtractComments_PrefersIDWhenBothPresent(t *testing.T) {
	raw := json.RawMessage(`{"id":"id-wins","comment_id":"comment-id-loses","text":"keep","from":{"id":"u1"},"media":{"id":"m1"}}`)

	payload := MetaPayload{
		Object: "instagram",
		Entry:  []MetaEntry{{ID: "e1", Changes: []MetaChange{{Field: "comments", Value: raw}}}},
	}

	comments := ExtractComments(payload)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Value.ID != "id-wins" {
		t.Errorf("expected id field to take priority, got %q", comments[0].Value.ID)
	}
}
