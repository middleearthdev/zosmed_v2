package seller

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// DefaultHoldSeconds is the reservation hold duration when no account-level
// setting overrides it. §8.1.4: default 300 s (5 minutes).
const DefaultHoldSeconds int32 = 300

// Sentinel errors returned by ReservationService methods.
var (
	// ErrOutOfStock is returned by Reserve when DecrementStock finds stock_left == 0.
	ErrOutOfStock = errors.New("seller: stok habis")

	// ErrProductNotFound is returned by Reserve when no product matches the code.
	ErrProductNotFound = errors.New("seller: produk tidak ditemukan untuk kode ini")
)

// reservationDB is the minimal database interface required by ReservationService.
// *dbgen.Queries satisfies this interface in production. An in-memory test double
// is used in unit tests so the state machine can be verified without Postgres.
type reservationDB interface {
	GetProductByPostAndCode(ctx context.Context, arg dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error)
	DecrementStock(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	IncrementStock(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	CreateReservation(ctx context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error)
	UpdateReservationStatus(ctx context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error)
}

// EnqueueExpireFunc enqueues a reservation:expire task after the given delay.
// reservationID is the UUID string. The concrete implementation in apps/worker
// wraps asynq.Client with asynq.ProcessIn(delay) + asynq.TaskID(reservationID)
// for idempotency (one timer per reservation, ADR-001 §2).
// Using a function type keeps libs/kits/seller free of an asynq import.
type EnqueueExpireFunc func(ctx context.Context, reservationID string, delay time.Duration) error

// ReservationService implements the comment-to-order state machine (ADR-001 §2).
// All transitions use UpdateReservationStatus with an expected_status guard to
// prevent lost-update races between concurrent asynq workers.
type ReservationService struct {
	db      reservationDB
	enqueue EnqueueExpireFunc
}

// NewReservationService returns a ReservationService backed by db and enqueue.
// enqueue is called once per successful Reserve to schedule the expiry task.
func NewReservationService(db reservationDB, enqueue EnqueueExpireFunc) *ReservationService {
	return &ReservationService{db: db, enqueue: enqueue}
}

// ReserveResult holds the outcome of a successful Reserve call.
type ReserveResult struct {
	Reservation dbgen.Reservation
	ProductName string
}

// Reserve creates a reservation for a detected keep/order code (ADR-001 §2, state: start → reserved).
//
// Steps:
//  1. GetProductByPostAndCode — locate the product; ErrProductNotFound if missing.
//  2. DecrementStock atomically (WHERE stock_left > 0); ErrOutOfStock if zero rows.
//  3. Build the wa.me link (stored in the reservation for private-reply action).
//  4. CreateReservation with status=reserved and expires_at=now+holdSeconds.
//  5. Enqueue TaskReservationExpire with the hold delay (idempotent via TaskID).
//
// holdSeconds <= 0 uses DefaultHoldSeconds.
// waPhone, fromUsername, code, and productName populate the wa.me prefill.
// {nama} ONLY from fromUsername (webhook payload) — never scraped (§4b.7).
func (s *ReservationService) Reserve(
	ctx context.Context,
	accountID pgtype.UUID,
	catalogPostID pgtype.UUID,
	code string,
	commentID string,
	fromID string,
	fromUsername string,
	waPhone string,
	holdSeconds int32,
) (ReserveResult, error) {
	if holdSeconds <= 0 {
		holdSeconds = DefaultHoldSeconds
	}

	// Step 1: find the product for this code on the post.
	product, err := s.db.GetProductByPostAndCode(ctx, dbgen.GetProductByPostAndCodeParams{
		CatalogPostID: catalogPostID,
		Code:          code,
	})
	if isNoRows(err) {
		return ReserveResult{}, ErrProductNotFound
	}
	if err != nil {
		return ReserveResult{}, fmt.Errorf("seller: get product: %w", err)
	}

	// Step 2: atomically claim one stock unit; zero rows means out of stock.
	_, err = s.db.DecrementStock(ctx, product.ID)
	if isNoRows(err) {
		return ReserveResult{}, ErrOutOfStock
	}
	if err != nil {
		return ReserveResult{}, fmt.Errorf("seller: decrement stock: %w", err)
	}

	// Step 3: build wa.me link before persisting.
	waLink := BuildWaLink(waPhone, fromUsername, code, product.Name)

	now := time.Now()
	expiresAt := now.Add(time.Duration(holdSeconds) * time.Second)

	// Step 4: persist the reservation.
	res, err := s.db.CreateReservation(ctx, dbgen.CreateReservationParams{
		AccountID:       accountID,
		CatalogPostID:   catalogPostID,
		ProductID:       product.ID,
		Code:            code,
		IgCommentID:     commentID,
		ContactIgUserID: fromID,
		ContactHandle:   fromUsername,
		HoldSeconds:     holdSeconds,
		ExpiresAt:       pgtype.Timestamptz{Time: expiresAt, Valid: true},
		WaLink:          waLink,
	})
	if err != nil {
		return ReserveResult{}, fmt.Errorf("seller: create reservation: %w", err)
	}

	// Step 5: enqueue expiry task. TaskID = reservationID ensures idempotency —
	// only one timer per reservation even on task retry.
	resID := UUIDToString(res.ID)
	if err := s.enqueue(ctx, resID, time.Duration(holdSeconds)*time.Second); err != nil {
		// Expiry enqueueing failure is non-fatal for the reservation itself, but
		// the hold may not be auto-released. Return the error so the caller can log.
		return ReserveResult{Reservation: res, ProductName: product.Name},
			fmt.Errorf("seller: enqueue expire (reservation created, timer missing): %w", err)
	}

	return ReserveResult{Reservation: res, ProductName: product.Name}, nil
}

// MarkWaitingPay transitions reserved → waiting-pay (ADR-001 §2).
// Must be called AFTER the private reply has been confirmed sent.
// Uses UpdateReservationStatus guard (expected=reserved) — race-safe.
func (s *ReservationService) MarkWaitingPay(ctx context.Context, reservationID string) error {
	id, err := ParseUUID(reservationID)
	if err != nil {
		return fmt.Errorf("seller: MarkWaitingPay: %w", err)
	}
	_, err = s.db.UpdateReservationStatus(ctx, dbgen.UpdateReservationStatusParams{
		NewStatus:      dbgen.ReservationStatusWaitingPay,
		ClosedAt:       pgtype.Timestamptz{Valid: false},
		ID:             id,
		ExpectedStatus: dbgen.ReservationStatusReserved,
	})
	if isNoRows(err) {
		// Guard fired: another worker already transitioned — treat as no-op.
		return nil
	}
	if err != nil {
		return fmt.Errorf("seller: MarkWaitingPay: %w", err)
	}
	return nil
}

// Close transitions waiting-pay → closed-wa (terminal success, ADR-001 §2).
// closed_at is set to now. Stock is NOT returned — item is sold.
func (s *ReservationService) Close(ctx context.Context, reservationID string) error {
	id, err := ParseUUID(reservationID)
	if err != nil {
		return fmt.Errorf("seller: Close: %w", err)
	}
	_, err = s.db.UpdateReservationStatus(ctx, dbgen.UpdateReservationStatusParams{
		NewStatus:      dbgen.ReservationStatusClosedWa,
		ClosedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ID:             id,
		ExpectedStatus: dbgen.ReservationStatusWaitingPay,
	})
	if isNoRows(err) {
		return nil // guard: already terminal
	}
	if err != nil {
		return fmt.Errorf("seller: Close: %w", err)
	}
	return nil
}

// Expire is called by the reservation:expire task handler (ADR-001 §2, §3.3).
// It is idempotent:
//   - status reserved or waiting-pay → IncrementStock + transition to expired-released.
//   - status already terminal (closed-wa, expired-released) → no-op (guard race).
//
// Stock is returned ONLY when transitioning from a non-terminal state.
func (s *ReservationService) Expire(ctx context.Context, reservationID string) error {
	id, err := ParseUUID(reservationID)
	if err != nil {
		return fmt.Errorf("seller: Expire: %w", err)
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	// Attempt reserved → expired-released.
	r, err := s.db.UpdateReservationStatus(ctx, dbgen.UpdateReservationStatusParams{
		NewStatus:      dbgen.ReservationStatusExpiredReleased,
		ClosedAt:       now,
		ID:             id,
		ExpectedStatus: dbgen.ReservationStatusReserved,
	})
	if err == nil {
		if _, err := s.db.IncrementStock(ctx, r.ProductID); err != nil {
			return fmt.Errorf("seller: Expire: increment stock (from reserved): %w", err)
		}
		return nil
	}
	if !isNoRows(err) {
		return fmt.Errorf("seller: Expire (reserved→expired): %w", err)
	}

	// Attempt waiting-pay → expired-released.
	r, err = s.db.UpdateReservationStatus(ctx, dbgen.UpdateReservationStatusParams{
		NewStatus:      dbgen.ReservationStatusExpiredReleased,
		ClosedAt:       now,
		ID:             id,
		ExpectedStatus: dbgen.ReservationStatusWaitingPay,
	})
	if err == nil {
		if _, err := s.db.IncrementStock(ctx, r.ProductID); err != nil {
			return fmt.Errorf("seller: Expire: increment stock (from waiting-pay): %w", err)
		}
		return nil
	}
	if !isNoRows(err) {
		return fmt.Errorf("seller: Expire (waiting-pay→expired): %w", err)
	}

	// Both guards fired → status is already terminal (closed-wa or expired-released).
	// No-op: this is the correct idempotent behaviour for duplicate expire tasks.
	return nil
}

// ── Exported UUID helpers ─────────────────────────────────────────────────────

// ParseUUID parses a hyphenated UUID string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// into a pgtype.UUID. Exported so apps/worker task handlers can use it without
// re-implementing UUID parsing (DRY §12a-1).
func ParseUUID(s string) (pgtype.UUID, error) {
	clean := strings.ReplaceAll(s, "-", "")
	b, err := hex.DecodeString(clean)
	if err != nil || len(b) != 16 {
		return pgtype.UUID{}, fmt.Errorf("seller: invalid UUID %q", s)
	}
	var arr [16]byte
	copy(arr[:], b)
	return pgtype.UUID{Bytes: arr, Valid: true}, nil
}

// UUIDToString formats a pgtype.UUID as a lowercase hyphenated UUID string.
// Exported for use by apps/worker handlers alongside ParseUUID.
func UUIDToString(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ── package-private helpers ───────────────────────────────────────────────────

// isNoRows returns true when err indicates "no rows in result set".
// We match on the error string rather than importing pgx to keep the seller kit's
// dependency graph minimal. This string is stable across pgx v5 minor versions.
func isNoRows(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "no rows in result set")
}
