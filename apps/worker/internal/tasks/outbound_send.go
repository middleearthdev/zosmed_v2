package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
	"github.com/zosmed/zosmed/libs/workflow"
)

// outboundStore reads the account (token) for a retry, and the reservation
// (status guard) when the retry is reservation-coupled (seller kit).
// *dbgen.Queries satisfies it.
type outboundStore interface {
	GetAccountByID(ctx context.Context, id pgtype.UUID) (dbgen.Account, error)
	GetReservation(ctx context.Context, id pgtype.UUID) (dbgen.Reservation, error)
}

// waitingPayMarker transitions reserved → waiting-pay after a successful send.
// *seller.ReservationService satisfies it.
type waitingPayMarker interface {
	MarkWaitingPay(ctx context.Context, reservationID string) error
}

// OutboundSender sends an outbound IG message by Kind (ADR-007 §3.6) — the
// same 3-method contract as workflow.Sender. *igapi.Client satisfies it; a
// fake is used in tests. Exported (as an interface, via SenderFactory) so
// apps/worker/cmd/worker/main.go can wire it without this package importing
// igapi directly in the interface declaration.
type OutboundSender interface {
	ReplyToComment(ctx context.Context, commentID, text string) error
	SendPrivateReply(ctx context.Context, igUserID, commentID, text string) error
	SendDM(ctx context.Context, igUserID, targetUserID, text string) error
}

// SenderFactory builds an OutboundSender from a per-account token (ADR-002
// §6.2 — token looked up per task, never held statically).
type SenderFactory func(token string) OutboundSender

// OutboundSendHandler handles the "outbound:send" task (ADR-007 §3/§3.6): a
// single, Kind-aware, segment-neutral retry for ANY outbound IG message the
// safety gate previously deferred (workflow.DecisionQueue) — private-reply,
// dm, or comment-reply alike. This absorbs the former seller-only handler
// (MAJOR-2): reservation coupling (seller kit) is now OPTIONAL, active only
// when the payload carries a non-empty ReservationID, so a deployment without
// the seller kit wired (a purely neutral workflow) stays correct. Every
// attempt re-passes through the safety gate (§10 one-door) before any igapi
// call, and is dropped — never sent late — once its §4c Deadline has passed.
type OutboundSendHandler struct {
	store     outboundStore
	gate      workflow.Gater
	marker    waitingPayMarker
	newSender SenderFactory
	logger    *slog.Logger
}

// NewOutboundSendHandler builds the handler. In production store=r.DB,
// gate=r.Gate, marker=r.Svc, newSender wraps igapi.New.
func NewOutboundSendHandler(store outboundStore, gate workflow.Gater, marker waitingPayMarker, newSender SenderFactory, logger *slog.Logger) *OutboundSendHandler {
	return &OutboundSendHandler{store: store, gate: gate, marker: marker, newSender: newSender, logger: logger}
}

// ProcessTask implements the asynq handler signature.
//
// Returning a non-nil error signals asynq to retry (bounded by MaxRetry set
// at enqueue). A nil return means "done, don't retry" — used for every
// terminal outcome (sent, dropped, rejected).
func (h *OutboundSendHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ptasks.OutboundSendPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("outbound_send: unmarshal: %w", err)
	}

	log := h.logger.With(
		slog.String("kind", p.Kind),
		slog.String("object_id", p.ObjectID),
		slog.String("reservation_id", p.ReservationID),
	)

	// §4c TTL (ADR-007 #3): a stale retry is DROPPED, never sent late. Checked
	// before touching the account/gate/igapi — cheapest possible short-circuit.
	deadline, err := time.Parse(time.RFC3339, p.Deadline)
	if err != nil {
		return fmt.Errorf("outbound_send: invalid deadline %q: %w", p.Deadline, err)
	}
	if time.Now().After(deadline) {
		log.Warn("outbound_send: deadline §4c lewat — drop (bukan kirim telat)")
		return nil
	}

	accountID, err := uuidx.Parse(p.AccountID)
	if err != nil {
		return fmt.Errorf("outbound_send: parse account_id %q: %w", p.AccountID, err)
	}

	// Load account for its per-account token (never from payload — ADR-002 §6.2).
	account, err := h.store.GetAccountByID(ctx, accountID)
	if err != nil {
		if isNoRows(err) {
			log.Warn("outbound_send: unknown account — drop")
			return nil
		}
		return fmt.Errorf("outbound_send: get account: %w", err)
	}
	if account.Status != "connected" {
		log.Warn("outbound_send: account not connected — drop", slog.String("status", account.Status))
		return nil
	}

	// Reservation coupling is OPTIONAL — only the seller kit's private-reply
	// path sets ReservationID (ADR-007 §2.1 point 4). Guard: only retry while
	// the reservation is still `reserved`; if a prior attempt already
	// succeeded (or it expired), do NOT send — prevents a duplicate private
	// reply. Absent entirely for every neutral node's deferred outbound.
	if p.ReservationID != "" {
		resID, err := uuidx.Parse(p.ReservationID)
		if err != nil {
			return fmt.Errorf("outbound_send: parse reservation_id %q: %w", p.ReservationID, err)
		}
		res, err := h.store.GetReservation(ctx, resID)
		if err != nil {
			if isNoRows(err) {
				log.Warn("outbound_send: reservation gone — drop")
				return nil
			}
			return fmt.Errorf("outbound_send: get reservation: %w", err)
		}
		if res.Status != dbgen.ReservationStatusReserved {
			log.Info("outbound_send: reservation no longer reserved — skip",
				slog.String("status", string(res.Status)))
			return nil
		}
	}

	// Re-check the safety gate (one-door §10) before any igapi call. A
	// missing/corrupt timestamp would leave commentAt zero, which the gate
	// treats as "window unenforceable → allow" (window.go) — i.e. fail-OPEN
	// past the §4c 7-day/24h window. Refuse instead of sending blind: a parse
	// failure is a retryable error; an explicit zero (the RFC3339 zero value
	// parses fine) is dropped for the window-bound kinds. comment-reply has no
	// hard §4c window — its Deadline TTL above already bounds it.
	commentAt, err := time.Parse(time.RFC3339, p.CommentAt)
	if err != nil {
		return fmt.Errorf("outbound_send: invalid comment_at %q: %w", p.CommentAt, err)
	}
	if commentAt.IsZero() && p.Kind != "comment-reply" {
		log.Warn("outbound_send: comment_at kosong untuk kind ber-window §4c — drop (tolak fail-open)")
		return nil
	}
	d, err := h.gate.Allow(ctx, workflow.OutboundReq{
		AccountID:    p.AccountID,
		Kind:         p.Kind,
		TargetUserID: p.TargetUserID,
		TriggerKey:   p.TriggerKey,
		CommentID:    p.ObjectID,
		CommentAt:    commentAt,
		PostID:       p.PostID,
	})
	if err != nil {
		return fmt.Errorf("outbound_send: gate: %w", err)
	}

	switch d.Action {
	case workflow.DecisionAllow:
		if err := sendByKind(ctx, h.newSender(account.AccessToken), p); err != nil {
			return fmt.Errorf("outbound_send: send: %w", err)
		}
		if p.ReservationID != "" && h.marker != nil {
			if err := h.marker.MarkWaitingPay(ctx, p.ReservationID); err != nil {
				// Reply was sent; the transition is non-fatal (reconcile/expire cope).
				log.Warn("outbound_send: sent but MarkWaitingPay failed", slog.String("error", err.Error()))
			}
		}
		log.Info("outbound_send: deferred outbound sent")
		return nil

	case workflow.DecisionQueue:
		// Still over quota — return an error so asynq retries (bounded by MaxRetry).
		log.Info("outbound_send: gate still queue — will retry", slog.String("reason", d.Reason))
		return fmt.Errorf("outbound_send: gate=queue (%s); retry", d.Reason)

	default: // DecisionReject
		// Window closed / dedupe / kill-switch — give up; no further retry.
		log.Warn("outbound_send: gate reject — drop", slog.String("reason", d.Reason))
		return nil
	}
}

// sendByKind dispatches to the OutboundSender method matching p.Kind
// (ADR-007 §3.6). Kind always originates from this codebase's own enqueue
// closures (libs/workflow/nodes, libs/kits/seller) — never external input —
// so an unrecognised value is a programming error, surfaced as a retryable
// error rather than silently dropped.
func sendByKind(ctx context.Context, sender OutboundSender, p ptasks.OutboundSendPayload) error {
	switch p.Kind {
	case "comment-reply":
		return sender.ReplyToComment(ctx, p.ObjectID, p.Text)
	case "private-reply":
		return sender.SendPrivateReply(ctx, p.IgUserID, p.ObjectID, p.Text)
	case "dm":
		return sender.SendDM(ctx, p.IgUserID, p.TargetUserID, p.Text)
	default:
		return fmt.Errorf("outbound_send: unknown kind %q", p.Kind)
	}
}
