package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	seller "github.com/zosmed/zosmed/libs/kits/seller"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/apps/worker/internal/runner"
)

// ReservationExpireHandler handles the "reservation:expire" task.
// It calls ReservationService.Expire which is fully idempotent:
//   - status reserved or waiting-pay → IncrementStock + transition to expired-released.
//   - status already terminal (closed-wa or expired-released) → no-op (guard race).
//
// The task is enqueued with asynq.TaskID(reservationID) so asynq deduplicates
// concurrent submissions for the same reservation (one timer per reservation,
// ADR-001 §2).
type ReservationExpireHandler struct {
	svc    *seller.ReservationService
	logger *slog.Logger
}

// NewReservationExpireHandler builds the expire handler from the shared runner.
// It uses runner.Svc which is the same ReservationService wired in runner.New —
// consistent DB access and no risk of double-constructing the service.
func NewReservationExpireHandler(r *runner.Runner, logger *slog.Logger) *ReservationExpireHandler {
	return &ReservationExpireHandler{svc: r.Svc, logger: logger}
}

// ProcessTask implements the asynq handler signature.
func (h *ReservationExpireHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ptasks.ReservationExpirePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("reservation_expire: unmarshal: %w", err)
	}

	log := h.logger.With(slog.String("reservation_id", p.ReservationID))

	if err := h.svc.Expire(ctx, p.ReservationID); err != nil {
		log.Error("reservation_expire: expire failed", slog.String("error", err.Error()))
		return fmt.Errorf("reservation_expire: %w", err)
	}

	log.Info("reservation_expire: processed (idempotent)")
	return nil
}
