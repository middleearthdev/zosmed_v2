package seller

import (
	"context"
	"fmt"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// Node key constants. Stable identifiers used in WorkflowDef and runner wiring.
// Defined once here; import in apps/worker/internal/runner rather than hardcoding.
const (
	// NodeKeyCommentTrigger fires when Source == "comment" (post/Reel only, §4b.5).
	NodeKeyCommentTrigger = "seller.comment"
	// NodeKeyReserve claims stock and creates a Reservation.
	NodeKeyReserve = "seller.reserve"
	// NodeKeyPrivateReply gate-checks and sends the wa.me private reply.
	NodeKeyPrivateReply = "seller.private-reply"
)

// Event.Raw keys written by the comment:ingest handler before Engine.Run.
// These constants prevent string drift between the handler and the actions.
const (
	RawKeyCatalogPostID = "catalog_post_id" // string UUID of the matched catalog_post
	RawKeyKode          = "kode"            // normalised keep code (e.g. "C1", "keep")
	RawKeyHoldSeconds   = "hold_seconds"    // int32 from account settings (0 → default)
	RawKeyIgUserID      = "ig_user_id"      // business account IG user ID (sender)
	RawKeyCommentAt     = "comment_at"      // time.Time of the comment for window checks
)

// Vars keys shared between reserve → private-reply actions via RunContext.Vars.
const (
	varReservationID = "reservation_id"
	varNama          = "nama"
	varKode          = "kode"
	varProduk        = "produk"
	varWaLink        = "wa_link"
	varIgUserID      = "ig_user_id"
)

// RegisterNodes registers all seller kit nodes into reg.
// Must be called before workflow.NewEngine. Engine does NOT import this package —
// nodes are injected via the Registry (dependency inversion, §6/§8).
//
// enqueueOutbound schedules a private-reply retry when the safety gate returns
// Queue (MAJOR-2); pass nil to disable retry (Queue is reported deferred only).
func RegisterNodes(reg *workflow.Registry, svc *ReservationService, waPhone string, enqueueOutbound EnqueueOutboundFunc) {
	reg.RegisterTrigger(NodeKeyCommentTrigger, &commentTrigger{})
	reg.RegisterAction(NodeKeyReserve, &reserveAction{svc: svc, waPhone: waPhone})
	reg.RegisterAction(NodeKeyPrivateReply, &privateReplyAction{svc: svc, enqueueOutbound: enqueueOutbound})
}

// ── commentTrigger ────────────────────────────────────────────────────────────

// commentTrigger fires for any event whose Source is "comment".
// The comment:ingest handler already filters to keep-code comments before calling
// Engine.Run, so this trigger acts as a type check only.
// Guardrail §4b.5: intentionally does NOT reference IG Live events.
type commentTrigger struct{}

func (commentTrigger) Match(_ context.Context, e workflow.Event) bool {
	return e.Source == workflow.SourceComment
}

// ── reserveAction ─────────────────────────────────────────────────────────────

// reserveAction reads context from rc.Event.Raw (set by comment:ingest), calls
// ReservationService.Reserve, and stores the result in rc.Vars for the next action.
//
// Expected rc.Event.Raw keys (see RawKey* constants):
//   - catalog_post_id : string UUID
//   - kode            : string (detected keep/order code)
//   - hold_seconds    : int32  (0 → DefaultHoldSeconds)
//   - ig_user_id      : string (IG user ID of the business account, for SendPrivateReply)
type reserveAction struct {
	svc     *ReservationService
	waPhone string
}

func (a *reserveAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	kode := rawString(rc.Event.Raw, RawKeyKode)
	catalogPostIDStr := rawString(rc.Event.Raw, RawKeyCatalogPostID)
	holdSeconds := rawInt32(rc.Event.Raw, RawKeyHoldSeconds)
	igUserID := rawString(rc.Event.Raw, RawKeyIgUserID)

	accountID, err := ParseUUID(rc.Event.AccountID)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("seller.reserve: parse account_id: %w", err)
	}
	catalogPostID, err := ParseUUID(catalogPostIDStr)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("seller.reserve: parse catalog_post_id: %w", err)
	}

	result, err := a.svc.Reserve(ctx,
		accountID,
		catalogPostID,
		kode,
		rc.Event.ObjectID, // IG comment ID
		rc.Event.FromID,
		rc.Event.FromUsername, // {nama} — from webhook payload only (§4b.7)
		a.waPhone,
		holdSeconds,
	)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("seller.reserve: %w", err)
	}

	resIDStr := UUIDToString(result.Reservation.ID)

	// Populate Vars for the private-reply action.
	rc.Vars[varReservationID] = resIDStr
	rc.Vars[varNama] = rc.Event.FromUsername
	rc.Vars[varKode] = kode
	rc.Vars[varProduk] = result.ProductName
	rc.Vars[varWaLink] = result.Reservation.WaLink
	rc.Vars[varIgUserID] = igUserID

	return workflow.ActionResult{
		Detail: fmt.Sprintf("reservation %s created (status=reserved)", resIDStr),
	}, nil
}

// ── privateReplyAction ────────────────────────────────────────────────────────

// privateReplyAction sends exactly ONE private reply containing the wa.me link.
//
// Guardrail A: one outbound per comment — private reply includes the wa.me link
// so there is no separate DM blast.
// Guardrail F: ALL outbound passes rc.Gate.Allow before touching igapi.
//
// Gate outcomes (ADR-001 §5):
//   - Allow  → SendPrivateReply + MarkWaitingPay (reserved → waiting-pay)
//   - Queue  → reservation stays reserved; overflow queue retries later
//   - Reject → reservation stays reserved until the expire task fires
type privateReplyAction struct {
	svc             *ReservationService
	enqueueOutbound EnqueueOutboundFunc
}

func (a *privateReplyAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	reservationID := rc.Vars[varReservationID]
	nama := rc.Vars[varNama]
	kode := rc.Vars[varKode]
	produk := rc.Vars[varProduk]
	waLink := rc.Vars[varWaLink]
	igUserID := rc.Vars[varIgUserID]

	// Build a single reply text that includes the wa.me link (one outbound, guardrail A).
	replyText := BuildPrivateReplyText("", nama, kode, produk, waLink)

	req := workflow.OutboundReq{
		AccountID:    rc.Event.AccountID,
		Kind:         "private-reply",
		TargetUserID: rc.Event.FromID,
		TriggerKey:   rc.Event.ObjectID, // comment ID — dedupe key per (account, user, comment)
		CommentID:    rc.Event.ObjectID,
		CommentAt:    rawTime(rc.Event.Raw, RawKeyCommentAt),
		PostID:       rc.Event.MediaID,
	}

	// Guardrail F: gate check BEFORE any igapi call.
	d, err := rc.Gate.Allow(ctx, req)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("seller.private-reply: gate: %w", err)
	}

	switch d.Action {
	case workflow.DecisionAllow:
		// Send the private reply (igapi call happens inside Sender — never direct).
		if err := rc.Sender.SendPrivateReply(ctx, igUserID, rc.Event.ObjectID, replyText); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("seller.private-reply: send: %w", err)
		}
		// Transition reserved → waiting-pay now that reply is confirmed sent.
		if err := a.svc.MarkWaitingPay(ctx, reservationID); err != nil {
			// Non-fatal: reply was sent. Log via step detail.
			return workflow.ActionResult{
				Detail: fmt.Sprintf("private reply sent; MarkWaitingPay err: %v", err),
			}, nil
		}
		return workflow.ActionResult{Detail: "private reply sent; status=waiting-pay"}, nil

	case workflow.DecisionQueue:
		// Overflow (§4c): reservation stays reserved and the private reply is
		// re-queued to send when quota recovers (MAJOR-2). Without an enqueue
		// func wired, fall back to reporting deferred only.
		if a.enqueueOutbound == nil {
			return workflow.ActionResult{
				Detail: fmt.Sprintf("gate=queue (%s); outbound deferred (no retry wired)", d.Reason),
			}, nil
		}
		retry := OutboundRetry{
			AccountID:     rc.Event.AccountID,
			IgUserID:      igUserID,
			CommentID:     rc.Event.ObjectID,
			TargetUserID:  rc.Event.FromID,
			ReservationID: reservationID,
			ReplyText:     replyText,
			PostID:        rc.Event.MediaID,
			TriggerKey:    rc.Event.ObjectID,
			CommentAt:     rawTime(rc.Event.Raw, RawKeyCommentAt),
		}
		if err := a.enqueueOutbound(ctx, retry, OutboundRetryDelay); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("seller.private-reply: enqueue outbound retry: %w", err)
		}
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=queue (%s); outbound retry enqueued", d.Reason),
		}, nil

	default: // DecisionReject
		// Reservation stays reserved until reservation:expire fires and releases stock.
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=reject (%s); reservation stays reserved until expire", d.Reason),
		}, nil
	}
}

// ── Raw map helpers ───────────────────────────────────────────────────────────

func rawString(raw map[string]any, key string) string {
	if v, ok := raw[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func rawInt32(raw map[string]any, key string) int32 {
	if v, ok := raw[key]; ok {
		switch n := v.(type) {
		case int32:
			return n
		case int:
			return int32(n)
		case int64:
			return int32(n)
		case float64:
			return int32(n)
		}
	}
	return 0
}

func rawTime(raw map[string]any, key string) time.Time {
	if v, ok := raw[key]; ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}
