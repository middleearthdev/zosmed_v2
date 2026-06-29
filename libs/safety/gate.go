package safety

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ── Outbound kind constants ───────────────────────────────────────────────────

// KindPrivateReply is a private reply to a comment. It is DELIVERED as a DM
// (POST /{ig-id}/messages, recipient {comment_id}) so it counts against the DM
// rate limits (§4c: 200/hr overflow→Queue, 1000/day), plus the ≤7-day window.
const KindPrivateReply = "private-reply"

// KindDM is a standard direct message (§4c: 24h window, 200/hr overflow→Queue).
const KindDM = "dm"

// KindCommentReply is a PUBLIC comment reply (POST /{comment-id}/replies, §7
// "Reply comment" action). It counts against the comment-reply caps (§4c: 750/hr,
// 30/post/5min) — NOT the DM caps. Distinct from KindPrivateReply, which is a DM.
const KindCommentReply = "comment-reply"

// ── OutboundReq ──────────────────────────────────────────────────────────────

// OutboundReq describes a single outbound IG message to be evaluated by Gate.
// All senders (private reply, DM) MUST populate this struct and call Gate.Allow
// before touching igapi. No field may be omitted from the mandatory set.
//
// Mandatory fields are fixed by contract (kits/seller and apps/worker import this).
// PostID is optional: when set, the per-post/5-min counter is also checked.
type OutboundReq struct {
	AccountID    string    // IG account sending the message
	Kind         string    // KindPrivateReply | KindDM
	TargetUserID string    // recipient's IG user ID
	TriggerKey   string    // dedupe key — e.g. commentID for private-reply, event key for DM
	CommentID    string    // source comment ID (required for private-reply; used for audit)
	CommentAt    time.Time // timestamp of the triggering interaction (for window checks)

	// PostID is optional. When non-empty, the commentsPerPostPer5min counter is
	// also evaluated. Leave empty if the caller cannot determine the post ID.
	PostID string
}

// ── Gate interface ────────────────────────────────────────────────────────────

// Gate is the outbound safety layer. Every IG sender MUST call Allow before
// sending. No outbound message may bypass this gate (CLAUDE.md §10 / §12).
type Gate interface {
	// Allow evaluates kill-switch → dedupe → window → quota in order.
	// Returns Allow only when all checks pass; counters and dedupe are set
	// atomically only on Allow (never on Queue or Reject).
	Allow(ctx context.Context, req OutboundReq) (Decision, error)

	// CurrentUsage returns live quota gauges for display in Safety Center UI
	// ("200/200 dm·hr", CLAUDE.md §10). Light read — no mutations.
	CurrentUsage(ctx context.Context, accountID string) ([]QuotaGauge, error)
}

// ── Option / config ───────────────────────────────────────────────────────────

// config holds gate options. Currently empty — caps come from constants.go.
// The Option pattern is in place for per-account cap overrides in a later phase
// without breaking the New() signature (§12a-4: no premature abstraction).
type config struct{}

func defaultConfig() config { return config{} }

// Option is a functional option for New().
type Option func(*config)

// ── Constructor ───────────────────────────────────────────────────────────────

// New creates a Gate backed by the given Redis client.
// Pass Option functions to override defaults (none defined yet).
func New(rdb redis.UniversalClient, opts ...Option) Gate {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return &gate{rdb: rdb, cfg: cfg}
}

// ── Concrete implementation ───────────────────────────────────────────────────

type gate struct {
	rdb redis.UniversalClient
	cfg config
}

// Allow evaluates the request through all safety checks in the mandated order:
//  1. Kill switch  → Reject if engaged
//  2. Dedupe       → Reject if (account, user, trigger) already sent
//  3. Window       → Reject if outside 7-day (private-reply) or 24h (DM) window
//  4. Quota        → Queue if at/near cap; Reject if kill-switch toggled mid-flight
//  5. On pass      → increment counters + mark dedupe atomically, return Allow
//
// Counters are NEVER incremented for Queue or Reject outcomes.
func (g *gate) Allow(ctx context.Context, req OutboundReq) (Decision, error) {
	// 1. Kill switch — global manual stop for an account.
	killed, err := g.isKillSwitchActive(ctx, req.AccountID)
	if err != nil {
		return Decision{}, fmt.Errorf("safety: kill-switch check: %w", err)
	}
	if killed {
		return Decision{Action: Reject, Reason: "kill-switch aktif"}, nil
	}

	// 2. Dedupe — block duplicate (account, user, trigger) outbound.
	dup, err := g.isDuplicate(ctx, req)
	if err != nil {
		return Decision{}, fmt.Errorf("safety: dedupe check: %w", err)
	}
	if dup {
		return Decision{Action: Reject, Reason: "dedupe: sudah pernah dikirim"}, nil
	}

	// 3. Window — enforce 7-day private-reply and 24h DM windows.
	if wd, ok := checkWindow(req); !ok {
		return wd, nil
	}

	// 4. Quota — rate-limit caps; DM overflow → Queue, not Reject.
	qd, err := g.checkQuota(ctx, req)
	if err != nil {
		return Decision{}, fmt.Errorf("safety: quota check: %w", err)
	}
	if qd.Action != Allow {
		return qd, nil
	}

	// 5. All checks passed — atomically increment counters and mark dedupe.
	pipe := g.rdb.Pipeline()
	g.incrementCounters(ctx, pipe, req)
	g.markDuplicate(ctx, pipe, req)
	if _, err := pipe.Exec(ctx); err != nil {
		return Decision{}, fmt.Errorf("safety: commit counters+dedupe: %w", err)
	}

	return Decision{Action: Allow, Reason: "ok"}, nil
}

// ── Kill switch ───────────────────────────────────────────────────────────────

const killSwitchKeyPrefix = "safety:killswitch:"

func killSwitchKey(accountID string) string {
	return killSwitchKeyPrefix + accountID
}

func (g *gate) isKillSwitchActive(ctx context.Context, accountID string) (bool, error) {
	_, err := g.rdb.Get(ctx, killSwitchKey(accountID)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("safety: kill-switch get: %w", err)
	}
	return true, nil
}

// EngageKillSwitch sets the global kill switch for an account. All subsequent
// Allow calls for that account return Reject until DisengageKillSwitch is called.
// This is the manual stop mechanism from the Safety Center UI (CLAUDE.md §10).
func EngageKillSwitch(ctx context.Context, rdb redis.UniversalClient, accountID string) error {
	// TTL = 0 → no expiry; persists until explicitly disengaged.
	if err := rdb.Set(ctx, killSwitchKey(accountID), "1", 0).Err(); err != nil {
		return fmt.Errorf("safety: engage kill-switch: %w", err)
	}
	return nil
}

// DisengageKillSwitch removes the kill switch for an account, resuming outbound.
func DisengageKillSwitch(ctx context.Context, rdb redis.UniversalClient, accountID string) error {
	if err := rdb.Del(ctx, killSwitchKey(accountID)).Err(); err != nil {
		return fmt.Errorf("safety: disengage kill-switch: %w", err)
	}
	return nil
}
