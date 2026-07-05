package seller

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

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

// ReservationDB is the minimal database interface required by ReservationService.
// *dbgen.Queries satisfies this interface in production. An in-memory test double
// is used in unit tests so the state machine can be verified without Postgres.
// Exported so TxRunner implementations (which receive a tx-bound instance) can
// name it (MAJOR-3a).
type ReservationDB interface {
	GetProductByPostAndCode(ctx context.Context, arg dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error)
	DecrementStock(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	IncrementStock(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	CreateReservation(ctx context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error)
	UpdateReservationStatus(ctx context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error)
}

// TxRunner runs fn inside a single database transaction, passing a ReservationDB
// bound to that transaction. On error (or panic) the transaction rolls back
// (MAJOR-3a: prevents stock leaking when DecrementStock succeeds but the
// subsequent CreateReservation fails). Production uses NewPgxTxRunner; unit
// tests use a pass-through runner (no real transaction).
type TxRunner interface {
	InTx(ctx context.Context, fn func(q ReservationDB) error) error
}

// pgxTxRunner is the production TxRunner backed by a pgx connection pool.
type pgxTxRunner struct{ pool *pgxpool.Pool }

// NewPgxTxRunner returns a TxRunner that opens a real pgx transaction per call.
func NewPgxTxRunner(pool *pgxpool.Pool) TxRunner { return pgxTxRunner{pool: pool} }

func (r pgxTxRunner) InTx(ctx context.Context, fn func(q ReservationDB) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seller: begin tx: %w", err)
	}
	// Rollback is a no-op once Commit has succeeded, so this defer is safe on
	// the happy path and guarantees rollback on any early return / panic.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(dbgen.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("seller: commit tx: %w", err)
	}
	return nil
}

// sequentialTxRunner runs fn directly against db WITHOUT a real transaction.
// Used by NewReservationService (unit tests / in-memory double) where atomicity
// is not modelled; production wires NewPgxTxRunner via NewReservationServiceTx.
type sequentialTxRunner struct{ db ReservationDB }

func (r sequentialTxRunner) InTx(ctx context.Context, fn func(q ReservationDB) error) error {
	return fn(r.db)
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
	db      ReservationDB
	tx      TxRunner
	enqueue EnqueueExpireFunc
}

// NewReservationService returns a ReservationService backed by db and enqueue,
// using a pass-through (non-transactional) TxRunner. Intended for unit tests
// with an in-memory double; production must use NewReservationServiceTx so the
// reserve critical section is atomic (MAJOR-3a).
func NewReservationService(db ReservationDB, enqueue EnqueueExpireFunc) *ReservationService {
	return NewReservationServiceTx(db, sequentialTxRunner{db: db}, enqueue)
}

// NewReservationServiceTx returns a ReservationService whose Reserve wraps
// DecrementStock + CreateReservation in the transaction provided by tx.
func NewReservationServiceTx(db ReservationDB, tx TxRunner, enqueue EnqueueExpireFunc) *ReservationService {
	return &ReservationService{db: db, tx: tx, enqueue: enqueue}
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

	// Step 3: build wa.me link before persisting.
	waLink := BuildWaLink(waPhone, fromUsername, code, product.Name)

	now := time.Now()
	expiresAt := now.Add(time.Duration(holdSeconds) * time.Second)

	// Steps 2+4 in ONE transaction (MAJOR-3a): claim one stock unit AND persist
	// the reservation atomically. If CreateReservation fails after DecrementStock,
	// the whole transaction rolls back so stock is never leaked.
	var res dbgen.Reservation
	txErr := s.tx.InTx(ctx, func(q ReservationDB) error {
		// Atomically claim one stock unit; zero rows means out of stock.
		if _, e := q.DecrementStock(ctx, product.ID); e != nil {
			if isNoRows(e) {
				return ErrOutOfStock
			}
			return fmt.Errorf("seller: decrement stock: %w", e)
		}
		r, e := q.CreateReservation(ctx, dbgen.CreateReservationParams{
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
		if e != nil {
			return fmt.Errorf("seller: create reservation: %w", e)
		}
		res = r
		return nil
	})
	if errors.Is(txErr, ErrOutOfStock) {
		return ReserveResult{}, ErrOutOfStock
	}
	if txErr != nil {
		return ReserveResult{}, txErr
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
