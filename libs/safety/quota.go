package safety

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// QuotaGauge is a live usage counter for one quota dimension.
// Mirrors packages/types/src/safety.ts QuotaGauge; the JSON mapping is done
// by the apps/api HTTP handler (SoC — this lib is transport-agnostic).
type QuotaGauge struct {
	Key   string // e.g. "dm-hr"  — matches TS QuotaGauge.key
	Label string
	Used  int64
	Cap   int64
	Unit  string
}

// ── Redis key helpers (fixed-window buckets) ──────────────────────────────────

// Fixed-window bucket: seconds since epoch divided by the window size gives a
// monotonically increasing bucket ID that changes exactly once per window.

func commentReplyHrKey(accountID string, t time.Time) string {
	bucket := t.Unix() / 3600
	return fmt.Sprintf("safety:q:%s:cr:hr:%d", accountID, bucket)
}

func dmHrKey(accountID string, t time.Time) string {
	bucket := t.Unix() / 3600
	return fmt.Sprintf("safety:q:%s:dm:hr:%d", accountID, bucket)
}

func dmDayKey(accountID string, t time.Time) string {
	bucket := t.Unix() / 86400
	return fmt.Sprintf("safety:q:%s:dm:day:%d", accountID, bucket)
}

// commentPostPer5minKey tracks public comment replies per post per 5-minute
// window (§4c: cap 30, human-paced). Requires req.PostID to be non-empty.
func commentPostPer5minKey(accountID, postID string, t time.Time) string {
	bucket := t.Unix() / 300
	return fmt.Sprintf("safety:q:%s:cp5:%s:%d", accountID, postID, bucket)
}

// ── Counter read ──────────────────────────────────────────────────────────────

func (g *gate) getCounter(ctx context.Context, key string) (int64, error) {
	v, err := g.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("safety: get counter %q: %w", key, err)
	}
	return v, nil
}

// ── Quota check (read-only — no mutation) ────────────────────────────────────

// checkQuota reads current counters and returns a non-Allow decision if any
// quota boundary is breached. Counters are NOT mutated here; increment happens
// only after all Gate checks pass (gate.go Allow ordering guarantee).
func (g *gate) checkQuota(ctx context.Context, req OutboundReq) (Decision, error) {
	now := time.Now()

	switch req.Kind {

	case KindCommentReply:
		// PUBLIC comment reply (POST /{comment-id}/replies).
		// 1. Comment replies per hour (Meta technical cap = 750).
		crCount, err := g.getCounter(ctx, commentReplyHrKey(req.AccountID, now))
		if err != nil {
			return Decision{}, err
		}
		if crCount >= capCommentRepliesPerHour {
			return Decision{Action: Queue, Reason: "kuota comment-reply/jam penuh (750), masuk antrian"}, nil
		}
		if float64(crCount) >= float64(capCommentRepliesPerHour)*AutoPauseThreshold {
			return Decision{Action: Queue, Reason: "auto-pause · rate near limit comment-reply/jam"}, nil
		}

		// 2. Comments per post per 5 min (human-paced cap = 30).
		if req.PostID != "" {
			p5Count, err := g.getCounter(ctx, commentPostPer5minKey(req.AccountID, req.PostID, now))
			if err != nil {
				return Decision{}, err
			}
			if p5Count >= capCommentsPerPostPer5min {
				return Decision{Action: Queue, Reason: "kuota komentar/post/5mnt penuh (30), masuk antrian"}, nil
			}
		}

	case KindPrivateReply, KindDM:
		// A private reply is delivered as a DM, so both meter the DM caps (§4c).
		// 1. DM per hour — overflow MUST be Queue, not Reject (§4c / §10).
		dmHrCount, err := g.getCounter(ctx, dmHrKey(req.AccountID, now))
		if err != nil {
			return Decision{}, err
		}
		if dmHrCount >= capDMPerHour {
			return Decision{Action: Queue, Reason: "kuota DM/jam penuh (200/hr), masuk antrian"}, nil
		}
		if float64(dmHrCount) >= float64(capDMPerHour)*AutoPauseThreshold {
			return Decision{Action: Queue, Reason: "auto-pause · rate near limit DM/jam"}, nil
		}

		// 2. DM per day (behaviour-based soft limit = 1 000).
		dmDayCount, err := g.getCounter(ctx, dmDayKey(req.AccountID, now))
		if err != nil {
			return Decision{}, err
		}
		if dmDayCount >= capDMPerDay {
			return Decision{Action: Queue, Reason: "kuota DM/hari penuh (1000/hari), masuk antrian"}, nil
		}
		if float64(dmDayCount) >= float64(capDMPerDay)*AutoPauseThreshold {
			return Decision{Action: Queue, Reason: "auto-pause · rate near limit DM/hari"}, nil
		}
	}

	return Decision{Action: Allow, Reason: "ok"}, nil
}

// ── Counter increment (called only on Allow) ──────────────────────────────────

// incrementCounters atomically increments all relevant quota counters for the
// outbound request using a Redis pipeline. It is pipelined together with
// markDuplicate so both succeed or fail as one unit.
func (g *gate) incrementCounters(ctx context.Context, pipe redis.Pipeliner, req OutboundReq) {
	now := time.Now()

	switch req.Kind {
	case KindCommentReply:
		crKey := commentReplyHrKey(req.AccountID, now)
		pipe.Incr(ctx, crKey)
		pipe.Expire(ctx, crKey, 2*time.Hour)

		if req.PostID != "" {
			p5Key := commentPostPer5minKey(req.AccountID, req.PostID, now)
			pipe.Incr(ctx, p5Key)
			pipe.Expire(ctx, p5Key, 10*time.Minute)
		}

	case KindPrivateReply, KindDM:
		// A private reply is delivered as a DM → meter DM counters (§4c).
		hrKey := dmHrKey(req.AccountID, now)
		pipe.Incr(ctx, hrKey)
		pipe.Expire(ctx, hrKey, 2*time.Hour)

		dayKey := dmDayKey(req.AccountID, now)
		pipe.Incr(ctx, dayKey)
		pipe.Expire(ctx, dayKey, 48*time.Hour)
	}
}

// ── CurrentUsage (Safety Center UI) ──────────────────────────────────────────

// CurrentUsage returns live quota gauges for the account. Used by the Safety
// Center screen to display "200/200 dm·hr" gauges (CLAUDE.md §10).
func (g *gate) CurrentUsage(ctx context.Context, accountID string) ([]QuotaGauge, error) {
	now := time.Now()

	crHr, err := g.getCounter(ctx, commentReplyHrKey(accountID, now))
	if err != nil {
		return nil, err
	}
	dmHr, err := g.getCounter(ctx, dmHrKey(accountID, now))
	if err != nil {
		return nil, err
	}
	dmDay, err := g.getCounter(ctx, dmDayKey(accountID, now))
	if err != nil {
		return nil, err
	}

	return []QuotaGauge{
		{
			Key:   "comment-replies-hr",
			Label: "Comment replies / jam",
			Used:  crHr,
			Cap:   capCommentRepliesPerHour,
			Unit:  "replies",
		},
		{
			Key:   "dm-hr",
			Label: "DM / jam",
			Used:  dmHr,
			Cap:   capDMPerHour,
			Unit:  "dm",
		},
		{
			Key:   "dm-day",
			Label: "DM / hari",
			Used:  dmDay,
			Cap:   capDMPerDay,
			Unit:  "dm",
		},
	}, nil
}
