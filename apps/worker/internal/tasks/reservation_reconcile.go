package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	seller "github.com/zosmed/zosmed/libs/kits/seller"
)

// reconcileBatchLimit bounds how many expired reservations one sweep processes,
// so a backlog can never turn a single tick into an unbounded transaction storm.
// Subsequent ticks drain the remainder.
const reconcileBatchLimit = 500

// expiredReservationLister returns active reservations already past expiry.
// *dbgen.Queries satisfies it; a fake is used in tests (§12a-4 — one concrete
// impl + a test double justify the seam).
type expiredReservationLister interface {
	ListExpiredActiveReservations(ctx context.Context, lim int32) ([]pgtype.UUID, error)
}

// reservationExpirer releases a reservation. *seller.ReservationService satisfies it.
type reservationExpirer interface {
	Expire(ctx context.Context, reservationID string) error
}

// ReservationReconcileHandler handles the periodic "reservation:reconcile" task
// (MAJOR-3b). It is a BACKSTOP for lost reservation:expire timers: if an expire
// task was never enqueued (enqueue failure) or was lost from Redis, the hold
// would otherwise persist forever. The sweep finds active reservations already
// past expires_at and runs the same idempotent Expire transition (release stock).
type ReservationReconcileHandler struct {
	lister  expiredReservationLister
	expirer reservationExpirer
	logger  *slog.Logger
}

// NewReservationReconcileHandler builds the reconcile handler. lister is r.DB and
// expirer is r.Svc in production (reuses the same wiring — no double construction).
func NewReservationReconcileHandler(lister expiredReservationLister, expirer reservationExpirer, logger *slog.Logger) *ReservationReconcileHandler {
	return &ReservationReconcileHandler{lister: lister, expirer: expirer, logger: logger}
}

// ProcessTask implements the asynq handler signature. Carries no payload — the
// due set is read fresh from Postgres each run.
func (h *ReservationReconcileHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	ids, err := h.lister.ListExpiredActiveReservations(ctx, reconcileBatchLimit)
	if err != nil {
		return fmt.Errorf("reservation_reconcile: list expired: %w", err)
	}

	released := 0
	for _, id := range ids {
		idStr := seller.UUIDToString(id)
		// Expire is idempotent: releases stock + transitions to expired-released,
		// and no-ops if the row is already terminal. Failure of one reservation
		// must not abort the whole sweep.
		if err := h.expirer.Expire(ctx, idStr); err != nil {
			h.logger.Error("reservation_reconcile: expire failed",
				slog.String("reservation_id", idStr),
				slog.String("error", err.Error()),
			)
			continue
		}
		released++
	}

	if len(ids) > 0 {
		h.logger.Info("reservation_reconcile: swept expired reservations",
			slog.Int("found", len(ids)),
			slog.Int("released", released),
		)
	}
	return nil
}
