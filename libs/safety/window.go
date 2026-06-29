package safety

import "time"

// checkWindow enforces Instagram messaging windows per §4c.
//
// Returns (Decision, true) when the request passes — caller should continue.
// Returns (Decision, false) when the request is rejected — caller should return
// the Decision immediately without incrementing any counters.
func checkWindow(req OutboundReq) (Decision, bool) {
	now := time.Now()

	switch req.Kind {
	case KindPrivateReply:
		// Private reply must be sent within PrivateReplyWindowDays of the comment.
		if req.CommentAt.IsZero() {
			// No timestamp supplied — cannot enforce; allow and let caller verify.
			return Decision{}, true
		}
		deadline := req.CommentAt.Add(time.Duration(PrivateReplyWindowDays) * 24 * time.Hour)
		if now.After(deadline) {
			return Decision{
				Action: Reject,
				Reason: "private-reply window 7d lewat",
			}, false
		}

	case KindDM:
		// Standard DM requires the user to have interacted within MessagingWindowHours.
		// CommentAt holds the timestamp of the triggering interaction (comment, story
		// reply, etc.). If zero, the caller is responsible for window tracking (e.g.
		// the Conversation.windowState stored in the database).
		if req.CommentAt.IsZero() {
			return Decision{}, true
		}
		deadline := req.CommentAt.Add(time.Duration(MessagingWindowHours) * time.Hour)
		if now.After(deadline) {
			return Decision{
				Action: Reject,
				Reason: "messaging window 24j lewat; arahkan ke opt-in",
			}, false
		}
	}

	return Decision{}, true
}
