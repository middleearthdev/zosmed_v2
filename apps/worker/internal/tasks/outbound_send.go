package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	seller "github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/workflow"
)

// outboundStore reads the account (token) and reservation (status guard) for a
// retry. *dbgen.Queries satisfies it.
type outboundStore interface {
	GetAccountByID(ctx context.Context, id pgtype.UUID) (dbgen.Account, error)
	GetReservation(ctx context.Context, id pgtype.UUID) (dbgen.Reservation, error)
}

// waitingPayMarker transitions reserved → waiting-pay after a successful send.
// *seller.ReservationService satisfies it.
type waitingPayMarker interface {
	MarkWaitingPay(ctx context.Context, reservationID string) error
}

// PrivateReplySender sends the private reply. *igapi.Client satisfies it; a fake
// is used in tests. Exported so the SenderFactory can be named from main.
type PrivateReplySender interface {
	SendPrivateReply(ctx context.Context, igUserID, objectID, text string) error
}

// SenderFactory builds a PrivateReplySender from a per-account token (ADR-002
// §6.2 — token looked up per task, never held statically).
type SenderFactory func(token string) PrivateReplySender

// OutboundSendHandler handles the "outbound:send" task (MAJOR-2): a retry for a
// private reply the safety gate previously deferred (Queue). The reservation was
// already created; this only re-attempts the private-reply step (idempotent —
// never re-reserves). Every attempt re-passes through the safety gate (§10 one-door).
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
// Returning a non-nil error signals asynq to retry (bounded by MaxRetry set at
// enqueue). A nil return means "done, don't retry" — used for the terminal
// outcomes (sent, rejected, reservation no longer eligible).
func (h *OutboundSendHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ptasks.OutboundSendPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("outbound_send: unmarshal: %w", err)
	}

	log := h.logger.With(
		slog.String("comment_id", p.CommentID),
		slog.String("reservation_id", p.ReservationID),
	)

	accountID, err := seller.ParseUUID(p.AccountID)
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

	// Guard: only retry while the reservation is still in `reserved`. If it has
	// already moved to waiting-pay/terminal (a prior attempt succeeded, or it
	// expired), do NOT send — this prevents a duplicate private reply.
	resID, err := seller.ParseUUID(p.ReservationID)
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

	// Re-check the safety gate (one-door §10) before any igapi call.
	// A missing/corrupt timestamp would leave commentAt zero, which the gate
	// treats as "window unenforceable → allow" (window.go) — i.e. fail-OPEN past
	// the §4c 7-day private-reply window. Refuse instead of sending blind.
	commentAt, err := time.Parse(time.RFC3339, p.CommentAt)
	if err != nil {
		return fmt.Errorf("outbound_send: invalid comment_at %q: %w", p.CommentAt, err)
	}
	d, err := h.gate.Allow(ctx, workflow.OutboundReq{
		AccountID:    p.AccountID,
		Kind:         "private-reply",
		TargetUserID: p.TargetUserID,
		TriggerKey:   p.TriggerKey,
		CommentID:    p.CommentID,
		CommentAt:    commentAt,
		PostID:       p.PostID,
	})
	if err != nil {
		return fmt.Errorf("outbound_send: gate: %w", err)
	}

	switch d.Action {
	case workflow.DecisionAllow:
		if err := h.newSender(account.AccessToken).SendPrivateReply(ctx, p.IgUserID, p.CommentID, p.ReplyText); err != nil {
			return fmt.Errorf("outbound_send: send: %w", err)
		}
		if err := h.marker.MarkWaitingPay(ctx, p.ReservationID); err != nil {
			// Reply was sent; the transition is non-fatal (reconcile/expire cope).
			log.Warn("outbound_send: sent but MarkWaitingPay failed", slog.String("error", err.Error()))
		}
		log.Info("outbound_send: deferred private reply sent; status=waiting-pay")
		return nil

	case workflow.DecisionQueue:
		// Still over quota — return an error so asynq retries (bounded by MaxRetry).
		log.Info("outbound_send: gate still queue — will retry", slog.String("reason", d.Reason))
		return fmt.Errorf("outbound_send: gate=queue (%s); retry", d.Reason)

	default: // DecisionReject
		// Window closed / kill-switch / dedupe — give up; reservation expires.
		log.Warn("outbound_send: gate reject — drop", slog.String("reason", d.Reason))
		return nil
	}
}
