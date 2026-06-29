// Package safety implements the outbound rate-limit and safety gate (CLAUDE.md §10).
// Every IG outbound sender MUST call Gate.Allow before touching igapi.
// No message leaves the system without passing through this layer.
package safety

// Rate-limit caps — defined ONCE here; mirror packages/types/src/constants.ts RATE_LIMITS.
// If you change these values, update the TS file in the same commit (§5a).
const (
	capCommentRepliesPerHour  int64 = 750
	capDMPerHour              int64 = 200
	capDMPerDay               int64 = 1000
	capCommentsPerPostPer5min int64 = 30
)

// AutoPauseThreshold is the usage fraction that triggers auto-pause (§10).
// Exported so callers can display the threshold in Safety Center UI.
const AutoPauseThreshold = 0.8

// PrivateReplyWindowDays — private reply must be sent within this many days of
// the original comment (§4c). Exported for window.go and caller reference.
const PrivateReplyWindowDays = 7

// MessagingWindowHours — standard DM is allowed only within this many hours of
// the user's last interaction (§4c). Exported for window.go and caller reference.
const MessagingWindowHours = 24
