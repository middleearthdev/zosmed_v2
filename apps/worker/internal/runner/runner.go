// Package runner wires the workflow engine, seller kit, safety gate, and igapi
// for the comment-to-order vertical slice (ADR-001 §3.3).
//
// One-door guarantee (guardrail F): every outbound IG call passes through the
// gateAdapter which wraps safety.Gate. igapi.Client is only called AFTER
// rc.Gate.Allow returns DecisionAllow inside the seller kit action.
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	seller "github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/safety"
	"github.com/zosmed/zosmed/libs/workflow"
)

// CommentToOrderWorkflow is the WorkflowDef for the seller kit comment-to-order
// flow: comment trigger → reserve stock → private reply with wa.me link.
var CommentToOrderWorkflow = workflow.WorkflowDef{
	ID:          "comment-to-order",
	TriggerKeys: []string{seller.NodeKeyCommentTrigger},
	// No engine-level filter: the comment:ingest handler pre-screens to
	// keep-code comments on registered catalog posts before Engine.Run.
	ActionKeys: []string{seller.NodeKeyReserve, seller.NodeKeyPrivateReply},
}

// Runner holds the wired workflow engine and shared services.
// The igapi.Client (Sender) is created per-call in task handlers using IGToken,
// keeping the Runner itself stateless with respect to the IG API.
type Runner struct {
	Engine  *workflow.Engine
	DB      *dbgen.Queries             // exposed for task handlers to load catalog/account context
	Gate    workflow.Gater             // safety.Gate adapted to workflow.Gater (one-door)
	Svc     *seller.ReservationService // exposed for reservation:expire handler
	IGToken string                     // IG page access token (MVP: single account from env)
}

// New creates a fully wired Runner.
//
//   - pool:        Postgres connection pool (from platform/db.New)
//   - rdb:         Redis client (for safety.Gate quota + dedupe counters)
//   - asynqClient: asynq client (for enqueueing reservation:expire tasks)
//   - waPhone:     WhatsApp phone number E.164 without '+' (e.g. "6281234567890")
//   - igToken:     IG page access token for the business account
func New(
	pool *pgxpool.Pool,
	rdb redis.UniversalClient,
	asynqClient *asynq.Client,
	waPhone string,
	igToken string,
) *Runner {
	db := dbgen.New(pool)

	// enqueueExpire wraps asynq.Client with idempotent options:
	//   asynq.TaskID(reservationID) — ensures only one timer per reservation
	//   asynq.ProcessIn(delay)      — schedules execution after hold duration
	enqueueExpire := seller.EnqueueExpireFunc(
		func(ctx context.Context, reservationID string, delay time.Duration) error {
			payload, err := json.Marshal(ptasks.ReservationExpirePayload{
				ReservationID: reservationID,
			})
			if err != nil {
				return fmt.Errorf("runner: marshal expire payload: %w", err)
			}
			task := asynq.NewTask(ptasks.TaskReservationExpire, payload,
				asynq.TaskID(reservationID), // idempotent: one timer per reservation
				asynq.ProcessIn(delay),
			)
			_, err = asynqClient.EnqueueContext(ctx, task)
			if err != nil {
				return fmt.Errorf("runner: enqueue expire: %w", err)
			}
			return nil
		},
	)

	svc := seller.NewReservationService(db, enqueueExpire)

	reg := workflow.NewRegistry()
	seller.RegisterNodes(reg, svc, waPhone)

	eng := workflow.NewEngine(reg, []workflow.WorkflowDef{CommentToOrderWorkflow})

	safetyGate := safety.New(rdb)
	adapted := &gateAdapter{g: safetyGate}

	return &Runner{
		Engine:  eng,
		DB:      db,
		Gate:    adapted,
		Svc:     svc,
		IGToken: igToken,
	}
}

// ── gateAdapter ───────────────────────────────────────────────────────────────

// gateAdapter adapts safety.Gate (libs/safety) to workflow.Gater (libs/workflow).
// Both OutboundReq types have identical fields; DecisionAction int values are
// cast directly since they are defined with the same iota ordering.
// This is the ~15-line adapter described in the spec (ADR-001 §3.3).
type gateAdapter struct {
	g safety.Gate
}

func (a *gateAdapter) Allow(ctx context.Context, req workflow.OutboundReq) (workflow.Decision, error) {
	safeReq := safety.OutboundReq{
		AccountID:    req.AccountID,
		Kind:         req.Kind,
		TargetUserID: req.TargetUserID,
		TriggerKey:   req.TriggerKey,
		CommentID:    req.CommentID,
		CommentAt:    req.CommentAt,
		PostID:       req.PostID,
	}
	d, err := a.g.Allow(ctx, safeReq)
	if err != nil {
		return workflow.Decision{}, err
	}
	return workflow.Decision{
		Action: workflow.DecisionAction(d.Action), // same iota: Allow=0, Queue=1, Reject=2
		Reason: d.Reason,
	}, nil
}
