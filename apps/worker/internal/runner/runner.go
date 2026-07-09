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
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/zosmed/zosmed/apps/worker/internal/wfload"
	seller "github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/safety"
	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

// outboundMaxRetry caps how many times a deferred outbound (ADR-007 §2.1) is
// retried by asynq before it is archived. The Deadline TTL (§4c) is the REAL
// stop: the handler drops a stale task at dequeue. MaxRetry only needs to be
// large enough that asynq's default exponential backoff (~n^4 seconds per
// attempt) spans the longest Deadline — cumulative wait at n=20 is ≈8 days,
// past the 7-day private-reply window — so an hourly quota bucket that stays
// saturated for most of an hour (the routine overflow case, §10) still gets
// retried after the bucket resets instead of archiving mid-wait. Beyond this
// a seller reservation is left to expire (reservation:expire /
// reservation:reconcile releases the stock).
const outboundMaxRetry = 20

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
// The igapi.Client (Sender) is created per-run in task handlers, built from
// the per-account token looked up via DB.GetAccountByID (ADR-002 §6.2) —
// there is no single static token anymore, so the Runner itself stays
// stateless with respect to IG credentials.
type Runner struct {
	// Engine runs the transitional fallback built-in comment-to-order workflow
	// (ADR-004 R3) for accounts that have not yet saved/activated an
	// equivalent workflow via the builder API. comment_ingest.go only falls
	// back to this when Loader.LoadLive returns zero `live` workflows.
	Engine *workflow.Engine
	// Compiler maps a persisted graph (loader.LoadLive) to a ready-to-run
	// (*workflow.Registry, workflow.WorkflowDef) pair using the FactoryMap
	// assembled in New (ADR-004 §1/§4).
	Compiler *workflow.Compiler
	Loader   *wfload.Loader   // reads `live` workflows for an account
	RunStore *wfload.RunStore // writes workflow_run rows (ADR-004 R2)

	DB   *dbgen.Queries             // exposed for task handlers to load catalog/account context
	Gate workflow.Gater             // safety.Gate adapted to workflow.Gater (one-door)
	Svc  *seller.ReservationService // exposed for reservation:expire handler
}

// New creates a fully wired Runner.
//
//   - pool:        Postgres connection pool (from platform/db.New)
//   - rdb:         Redis client (for safety.Gate quota + dedupe counters)
//   - asynqClient: asynq client (for enqueueing reservation:expire tasks)
//   - waPhone:     WhatsApp phone number E.164 without '+' (e.g. "6281234567890")
func New(
	pool *pgxpool.Pool,
	rdb redis.UniversalClient,
	asynqClient *asynq.Client,
	waPhone string,
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

	// Reserve runs DecrementStock + CreateReservation in one pgx transaction
	// (MAJOR-3a) — NewPgxTxRunner rolls back if creation fails so stock never leaks.
	svc := seller.NewReservationServiceTx(db, seller.NewPgxTxRunner(pool), enqueueExpire)

	// enqueueDeferred schedules a generic outbound:send retry when the safety
	// gate defers a send (DecisionQueue overflow, ADR-007 §2.1/§3.8). Shared by
	// every neutral action node (nodes.RegisterFactories) AND the seller kit
	// (seller.RegisterFactories) — one closure, one task, no per-Kit duplicate
	// (§12a-1). TaskID = "outbound:{account}:{kind}:{trigger}" keeps enqueue
	// idempotent per (account, kind, trigger) — re-enqueuing the identical
	// retry (e.g. a second DecisionQueue for the same event) collapses onto
	// the same asynq task instead of stacking duplicates.
	enqueueDeferred := nodes.EnqueueDeferredFunc(
		func(ctx context.Context, d nodes.DeferredOutbound, delay time.Duration) error {
			payload, err := json.Marshal(ptasks.OutboundSendPayload{
				AccountID:     d.AccountID,
				Kind:          d.Kind,
				IgUserID:      d.IgUserID,
				ObjectID:      d.ObjectID,
				TargetUserID:  d.TargetUserID,
				Text:          d.Text,
				PostID:        d.PostID,
				TriggerKey:    d.TriggerKey,
				CommentAt:     d.CommentAt.Format(time.RFC3339),
				Deadline:      d.Deadline.Format(time.RFC3339),
				ReservationID: d.ReservationID,
			})
			if err != nil {
				return fmt.Errorf("runner: marshal outbound payload: %w", err)
			}
			task := asynq.NewTask(ptasks.TaskOutboundSend, payload,
				asynq.TaskID("outbound:"+d.AccountID+":"+d.Kind+":"+d.TriggerKey),
				asynq.ProcessIn(delay),
				asynq.MaxRetry(outboundMaxRetry),
			)
			if _, err := asynqClient.EnqueueContext(ctx, task); err != nil {
				// A TaskID collision means the identical retry is already
				// scheduled (e.g. two same-Kind deferrals for one trigger) —
				// idempotent success, mirroring apps/api enqueue ingest.
				if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
					return nil
				}
				return fmt.Errorf("runner: enqueue outbound: %w", err)
			}
			return nil
		},
	)

	reg := workflow.NewRegistry()
	seller.RegisterNodes(reg, svc, waPhone, enqueueDeferred)

	eng := workflow.NewEngine(reg, []workflow.WorkflowDef{CommentToOrderWorkflow})

	safetyGate := safety.New(rdb)
	adapted := &gateAdapter{g: safetyGate}

	// FactoryMap assembly (ADR-004 §1 STARTUP diagram): neutral nodes first,
	// then the seller Kit — order doesn't matter, keys (node_type strings)
	// never collide between the two. Neither libs/workflow/nodes nor
	// libs/workflow/compile.go ever imports libs/kits/seller (§9 guardrail);
	// only this apps/worker wiring layer is allowed to know about both.
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, enqueueDeferred)
	seller.RegisterFactories(fmap, svc, waPhone, enqueueDeferred)
	compiler := workflow.NewCompiler(fmap)

	return &Runner{
		Engine:   eng,
		Compiler: compiler,
		Loader:   wfload.NewLoader(db),
		RunStore: wfload.NewRunStore(db),
		DB:       db,
		Gate:     adapted,
		Svc:      svc,
	}
}

// ── gateAdapter ───────────────────────────────────────────────────────────────

// gateAdapter adapts safety.Gate (libs/safety) to workflow.Gater (libs/workflow).
// Both OutboundReq types have identical fields; the verdict enums are mapped
// explicitly (M6) rather than cast, so the two packages' iota orderings can
// drift independently without silently mis-mapping a decision.
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
	return workflow.Decision{Action: mapDecisionAction(d.Action), Reason: d.Reason}, nil
}

// mapDecisionAction maps a safety verdict to the workflow enum explicitly. An
// unknown value fails safe to DecisionReject (never send on an unrecognised verdict).
func mapDecisionAction(a safety.Action) workflow.DecisionAction {
	switch a {
	case safety.Allow:
		return workflow.DecisionAllow
	case safety.Queue:
		return workflow.DecisionQueue
	case safety.Reject:
		return workflow.DecisionReject
	default:
		return workflow.DecisionReject
	}
}
