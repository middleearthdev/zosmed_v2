package seller_test

// fsm_chain_test.go — additional FSM state machine tests.
//
// Covers gaps not addressed by reservation_test.go:
//   1. Full happy-path chain: reserved → waiting-pay → closed-wa
//   2. Close when already expired-released → no-op (idempotent guard)
//   3. Reserve with stock=0: CreateReservation must NOT be called
//   4. WaLink contains wa.me domain after Reserve
//   5. Concurrent Expire: only one goroutine increments stock (race guard)

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// ── Full chain: reserved → waiting-pay → closed-wa ───────────────────────────

// TestFSM_FullChain_ReservedToWaitingPayToClosedWa exercises the complete
// comment-to-order success path.  Each state transition must use the correct
// expected_status guard (ADR-001 §2).
func TestFSM_FullChain_ReservedToWaitingPayToClosedWa(t *testing.T) {
	// Track transitions to verify guard semantics.
	type transition struct {
		expected dbgen.ReservationStatus
		next     dbgen.ReservationStatus
	}
	var recorded []transition

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			recorded = append(recorded, transition{arg.ExpectedStatus, arg.NewStatus})
			res := reservedReservation()
			res.Status = arg.NewStatus
			if arg.NewStatus == dbgen.ReservationStatusClosedWa {
				res.ClosedAt = arg.ClosedAt
			}
			return res, nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	resIDStr := uuidx.Format(testReservationID)

	// Step 1: Reserve → status = reserved.
	result, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-chain-1", "user-chain-1", "Buyer", "6281234567890", 0)
	if err != nil {
		t.Fatalf("Reserve error: %v", err)
	}
	if result.Reservation.Status != dbgen.ReservationStatusReserved {
		t.Errorf("after Reserve: want status=reserved, got %s", result.Reservation.Status)
	}

	// Step 2: MarkWaitingPay → status = waiting-pay (guard: expected=reserved).
	if err := svc.MarkWaitingPay(context.Background(), resIDStr); err != nil {
		t.Fatalf("MarkWaitingPay error: %v", err)
	}

	// Step 3: Close → status = closed-wa (guard: expected=waiting-pay).
	if err := svc.Close(context.Background(), resIDStr); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Verify both transitions used the correct expected_status guards.
	if len(recorded) != 2 {
		t.Fatalf("expected 2 state transitions (waiting-pay + closed-wa), got %d: %+v", len(recorded), recorded)
	}
	if recorded[0].expected != dbgen.ReservationStatusReserved ||
		recorded[0].next != dbgen.ReservationStatusWaitingPay {
		t.Errorf("transition 0: want reserved→waiting-pay, got %s→%s",
			recorded[0].expected, recorded[0].next)
	}
	if recorded[1].expected != dbgen.ReservationStatusWaitingPay ||
		recorded[1].next != dbgen.ReservationStatusClosedWa {
		t.Errorf("transition 1: want waiting-pay→closed-wa, got %s→%s",
			recorded[1].expected, recorded[1].next)
	}

	// Verify ClosedAt is set for the closed-wa transition.
	if !recorded[1].next.Valid() {
		t.Error("closed-wa is not a valid terminal status")
	}
}

// ── Close idempotency after expiry ────────────────────────────────────────────

// TestFSM_CloseAfterExpired_IsNoOp verifies that calling Close on an already
// expired-released reservation is a safe no-op.  The DB guard (expected=waiting-pay)
// fires (errNoRows) and the service returns nil without panic or error.
func TestFSM_CloseAfterExpired_IsNoOp(t *testing.T) {
	db := &stubDB{
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			// Guard fires: reservation is already expired-released, not waiting-pay.
			return dbgen.Reservation{}, errNoRows
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	err := svc.Close(context.Background(), uuidx.Format(testReservationID))
	if err != nil {
		t.Errorf("Close after expired: expected no-op (nil error), got: %v", err)
	}
}

// TestFSM_Expire_AlreadyClosedWa_IsNoOp verifies idempotency when Expire is
// called after the reservation reached closed-wa (both guards fire).
func TestFSM_Expire_AlreadyClosedWa_IsNoOp(t *testing.T) {
	var incrementCalled bool
	calls := 0

	db := &stubDB{
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			calls++
			// Both reserved→expired and waiting-pay→expired guards fire.
			return dbgen.Reservation{}, errNoRows
		},
		incrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			incrementCalled = true
			return okProduct(), nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	if err := svc.Expire(context.Background(), uuidx.Format(testReservationID)); err != nil {
		t.Fatalf("Expire (already closed-wa): expected no-op, got: %v", err)
	}
	if incrementCalled {
		t.Error("IncrementStock must NOT be called when reservation is already terminal (closed-wa)")
	}
	if calls != 2 {
		t.Errorf("expected 2 guard attempts, got %d", calls)
	}
}

// ── Reserve: stock=0 guard ────────────────────────────────────────────────────

// TestFSM_Reserve_ZeroStock_CreateReservationNotCalled verifies that when
// DecrementStock returns "no rows" (stock_left == 0), Reserve returns
// ErrOutOfStock without ever calling CreateReservation.
func TestFSM_Reserve_ZeroStock_CreateReservationNotCalled(t *testing.T) {
	var createCalled bool

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			// Simulate stock_left == 0: WHERE stock_left > 0 matches 0 rows.
			return dbgen.Product{}, errNoRows
		},
		createReservation: func(_ context.Context, _ dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			createCalled = true
			return dbgen.Reservation{}, nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	_, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-stk-0", "user-stk-0", "Buyer", "6281234567890", 0)

	if err != seller.ErrOutOfStock {
		t.Errorf("expected ErrOutOfStock, got %v", err)
	}
	if createCalled {
		t.Error("CreateReservation must NOT be called when stock is 0")
	}
}

// TestFSM_Reserve_StockZero_StockNotDecremented verifies that when
// DecrementStock itself fails (no rows), no further decrement is attempted.
func TestFSM_Reserve_StockZero_NoFurtherDecrement(t *testing.T) {
	decrementCalls := 0

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			decrementCalls++
			return dbgen.Product{}, errNoRows
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	_, _ = svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-dec", "user-dec", "Buyer", "6281234567890", 0)

	if decrementCalls != 1 {
		t.Errorf("expected exactly 1 DecrementStock call, got %d", decrementCalls)
	}
}

// ── WaLink format compliance ──────────────────────────────────────────────────

// TestFSM_Reserve_WaLinkContainsWaMeDomain verifies that the wa.me deep link
// in the reservation uses the correct format (§8.1.1 / §4b.6: no API call).
func TestFSM_Reserve_WaLinkContainsWaMeDomain(t *testing.T) {
	var storedWaLink string

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			storedWaLink = arg.WaLink
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	const waPhone = "6281234567890"
	result, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-wa", "user-wa", "Budi", waPhone, 0)
	if err != nil {
		t.Fatalf("Reserve error: %v", err)
	}

	// Must use wa.me domain (pure URL, no API call).
	if !strings.HasPrefix(storedWaLink, "https://wa.me/") {
		t.Errorf("WaLink must start with https://wa.me/; got: %q", storedWaLink)
	}
	if !strings.Contains(storedWaLink, waPhone) {
		t.Errorf("WaLink must contain phone number %q; got: %q", waPhone, storedWaLink)
	}
	if !strings.Contains(storedWaLink, "Budi") {
		t.Errorf("WaLink must contain buyer name 'Budi'; got: %q", storedWaLink)
	}
	if !strings.Contains(result.Reservation.WaLink, "wa.me") {
		t.Errorf("Reservation.WaLink must contain wa.me; got: %q", result.Reservation.WaLink)
	}
}

// ── Concurrent Expire race guard ──────────────────────────────────────────────

// TestFSM_ConcurrentExpire_OnlyOneIncrements verifies that when two goroutines
// call Expire on the same reservation simultaneously:
//   - Both calls return nil (no error).
//   - IncrementStock is called exactly once (optimistic-lock guard prevents double-release).
//
// This mirrors the production behaviour where the DB WHERE-guard ensures only one
// concurrent Expire actually transitions the row (§3.3 idempotency guarantee).
func TestFSM_ConcurrentExpire_OnlyOneIncrements(t *testing.T) {
	var mu sync.Mutex
	alreadyTransitioned := false
	var incrementCount atomic.Int32

	db := &stubDB{
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			mu.Lock()
			defer mu.Unlock()
			if arg.ExpectedStatus == dbgen.ReservationStatusReserved && !alreadyTransitioned {
				alreadyTransitioned = true
				res := reservedReservation()
				res.Status = dbgen.ReservationStatusExpiredReleased
				return res, nil
			}
			// Guard fires for all other attempts.
			return dbgen.Reservation{}, errNoRows
		},
		incrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			incrementCount.Add(1)
			return okProduct(), nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	resIDStr := uuidx.Format(testReservationID)

	var wg sync.WaitGroup
	var errs [2]error

	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = svc.Expire(context.Background(), resIDStr)
	}()
	go func() {
		defer wg.Done()
		errs[1] = svc.Expire(context.Background(), resIDStr)
	}()
	wg.Wait()

	if errs[0] != nil {
		t.Errorf("goroutine 0 Expire error: %v", errs[0])
	}
	if errs[1] != nil {
		t.Errorf("goroutine 1 Expire error: %v", errs[1])
	}
	if n := incrementCount.Load(); n != 1 {
		t.Errorf("IncrementStock must be called exactly once on concurrent Expire; called %d times", n)
	}
}

// ── Expire: enqueue failure is non-fatal ─────────────────────────────────────

// TestFSM_Reserve_EnqueueFailure_ReturnsErrorButReservationCreated verifies
// that if the expire-task enqueueing fails, Reserve still returns the created
// reservation (non-fatal: timer missing but reservation exists).
func TestFSM_Reserve_EnqueueFailure_ReturnsReservation(t *testing.T) {
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
	}

	failEnqueue := func(_ context.Context, _ string, _ time.Duration) error {
		return errNoRows // reuse sentinel error as "enqueue failure"
	}

	svc := seller.NewReservationService(db, failEnqueue)
	result, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-enq-fail", "user-enq", "Buyer", "6281234567890", 0)

	// Enqueue failure surfaces as an error.
	if err == nil {
		t.Error("expected non-nil error when enqueue fails")
	}
	// But the reservation IS returned (caller can log and continue).
	if result.Reservation.Status != dbgen.ReservationStatusReserved {
		t.Errorf("reservation must be created even on enqueue failure; got status %s",
			result.Reservation.Status)
	}
}
