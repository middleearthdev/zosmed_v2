package safety

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Dedupe TTLs align with the messaging window for each kind.
// After TTL expires the same (account, user, trigger) can be re-sent,
// which is correct: a new interaction re-opens the window.
const (
	dedupeTTLPrivateReply = time.Duration(PrivateReplyWindowDays) * 24 * time.Hour
	dedupeTTLDM           = time.Duration(MessagingWindowHours) * time.Hour
)

// dedupeKey returns the Redis key for a (kind, account, user, trigger) tuple.
// Kind is included so distinct outbound kinds (e.g. comment-reply vs
// private-reply) triggered by the same underlying event (same comment) do
// NOT collide on a single dedupe key — each kind gets its own idempotency
// slot. See ADR-007 §2.3a.
func dedupeKey(kind, accountID, targetUserID, triggerKey string) string {
	return fmt.Sprintf("safety:dedupe:%s:%s:%s:%s", kind, accountID, targetUserID, triggerKey)
}

// dedupeTTLFor returns the appropriate TTL for the given outbound kind.
func dedupeTTLFor(kind string) time.Duration {
	if kind == KindPrivateReply {
		return dedupeTTLPrivateReply
	}
	return dedupeTTLDM
}

// isDuplicate returns true if (account, user, trigger) was already sent
// within its TTL window. This is the outbound-layer dedupe — distinct from
// the ingest-layer comment_id dedupe (different purpose, same principle).
func (g *gate) isDuplicate(ctx context.Context, req OutboundReq) (bool, error) {
	key := dedupeKey(req.Kind, req.AccountID, req.TargetUserID, req.TriggerKey)
	_, err := g.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("safety: dedupe get %q: %w", key, err)
	}
	return true, nil
}

// markDuplicate records that (account, user, trigger) was sent.
// MUST only be called after all Gate checks pass (Allow outcome).
// Combined in a pipeline with incrementCounters for atomicity.
func (g *gate) markDuplicate(ctx context.Context, pipe redis.Pipeliner, req OutboundReq) {
	key := dedupeKey(req.Kind, req.AccountID, req.TargetUserID, req.TriggerKey)
	ttl := dedupeTTLFor(req.Kind)
	pipe.Set(ctx, key, "1", ttl)
}
