package seller_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// ── In-memory test double ─────────────────────────────────────────────────────

// errNoRows mimics pgx.ErrNoRows via the string suffix that isNoRows checks.
var errNoRows = errors.New("no rows in result set")

func mustUUID(s string) pgtype.UUID {
	u, err := seller.ParseUUID(s)
	if err != nil {
		panic(fmt.Sprintf("mustUUID(%q): %v", s, err))
	}
	return u
}

var (
	testAccountID     = mustUUID("aaaaaaaa-0000-0000-0000-000000000001")
	testCatalogPostID = mustUUID("bbbbbbbb-0000-0000-0000-000000000002")
	testProductID     = mustUUID("cccccccc-0000-0000-0000-000000000003")
	testReservationID = mustUUID("dddddddd-0000-0000-0000-000000000004")
)

// stubDB implements the reservationDB interface used by ReservationService.
// All methods are set as fields so individual tests can customise behaviour.
type stubDB struct {
	getProduct           func(ctx context.Context, arg dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error)
	decrementStock       func(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	incrementStock       func(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	createReservation    func(ctx context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error)
	updateResStatus      func(ctx context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error)
}

func (s *stubDB) GetProductByPostAndCode(ctx context.Context, arg dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
	return s.getProduct(ctx, arg)
}
func (s *stubDB) DecrementStock(ctx context.Context, id pgtype.UUID) (dbgen.Product, error) {
	return s.decrementStock(ctx, id)
}
func (s *stubDB) IncrementStock(ctx context.Context, id pgtype.UUID) (dbgen.Product, error) {
	return s.incrementStock(ctx, id)
}
func (s *stubDB) CreateReservation(ctx context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
	return s.createReservation(ctx, arg)
}
func (s *stubDB) UpdateReservationStatus(ctx context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
	return s.updateResStatus(ctx, arg)
}

// okProduct returns a product stub with stock_left > 0.
func okProduct() dbgen.Product {
	return dbgen.Product{ID: testProductID, CatalogPostID: testCatalogPostID,
		Code: "C1", Name: "Kaos Ungu M", PriceIdr: 189000, StockTotal: 10, StockLeft: 5}
}

// reservedReservation returns a stub Reservation in reserved state.
func reservedReservation() dbgen.Reservation {
	return dbgen.Reservation{
		ID:            testReservationID,
		AccountID:     testAccountID,
		CatalogPostID: testCatalogPostID,
		ProductID:     testProductID,
		Code:          "C1",
		Status:        dbgen.ReservationStatusReserved,
		HoldSeconds:   300,
		ReservedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(5 * time.Minute), Valid: true},
		WaLink:        "https://wa.me/628xxx?text=...",
	}
}

// noopEnqueue is an EnqueueExpireFunc that always succeeds without side effects.
func noopEnqueue(_ context.Context, _ string, _ time.Duration) error { return nil }

// ── Reserve tests ─────────────────────────────────────────────────────────────

func TestReserve_Success(t *testing.T) {
	var enqueuedID string
	var enqueueDelay time.Duration

	db := &stubDB{
		getProduct:     func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) { return okProduct(), nil },
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) { return okProduct(), nil },
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
	}

	svc := seller.NewReservationService(db, func(_ context.Context, id string, d time.Duration) error {
		enqueuedID = id
		enqueueDelay = d
		return nil
	})

	result, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-1", "user-1", "Budi", "6281234567890", 0)

	if err != nil {
		t.Fatalf("Reserve unexpected error: %v", err)
	}
	if result.Reservation.Status != dbgen.ReservationStatusReserved {
		t.Errorf("expected status=reserved, got %s", result.Reservation.Status)
	}
	if result.ProductName != "Kaos Ungu M" {
		t.Errorf("expected ProductName=Kaos Ungu M, got %q", result.ProductName)
	}
	if enqueuedID == "" {
		t.Error("expected expire task to be enqueued")
	}
	wantDelay := time.Duration(seller.DefaultHoldSeconds) * time.Second
	if enqueueDelay != wantDelay {
		t.Errorf("enqueue delay=%v, want %v", enqueueDelay, wantDelay)
	}
}

func TestReserve_OutOfStock(t *testing.T) {
	db := &stubDB{
		getProduct:     func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) { return okProduct(), nil },
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) { return dbgen.Product{}, errNoRows },
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	_, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-2", "user-2", "Sari", "6281234567890", 0)

	if !errors.Is(err, seller.ErrOutOfStock) {
		t.Errorf("expected ErrOutOfStock, got %v", err)
	}
}

func TestReserve_ProductNotFound(t *testing.T) {
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return dbgen.Product{}, errNoRows
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	_, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C9",
		"comment-3", "user-3", "Andi", "6281234567890", 0)

	if !errors.Is(err, seller.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

// ── MarkWaitingPay tests ──────────────────────────────────────────────────────

func TestMarkWaitingPay_ReservedToWaitingPay(t *testing.T) {
	var calledWith dbgen.UpdateReservationStatusParams

	db := &stubDB{
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			calledWith = arg
			res := reservedReservation()
			res.Status = dbgen.ReservationStatusWaitingPay
			return res, nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	resIDStr := seller.UUIDToString(testReservationID)

	if err := svc.MarkWaitingPay(context.Background(), resIDStr); err != nil {
		t.Fatalf("MarkWaitingPay error: %v", err)
	}
	if calledWith.NewStatus != dbgen.ReservationStatusWaitingPay {
		t.Errorf("expected new_status=waiting-pay, got %s", calledWith.NewStatus)
	}
	if calledWith.ExpectedStatus != dbgen.ReservationStatusReserved {
		t.Errorf("expected expected_status=reserved, got %s", calledWith.ExpectedStatus)
	}
}

func TestMarkWaitingPay_NoOpWhenGuardFires(t *testing.T) {
	// Guard fires (concurrent worker won the race) → no-op, no error.
	db := &stubDB{
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			return dbgen.Reservation{}, errNoRows
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	if err := svc.MarkWaitingPay(context.Background(), seller.UUIDToString(testReservationID)); err != nil {
		t.Errorf("expected no-op, got error: %v", err)
	}
}

// ── Close tests ───────────────────────────────────────────────────────────────

func TestClose_WaitingPayToClosedWa(t *testing.T) {
	var calledWith dbgen.UpdateReservationStatusParams

	db := &stubDB{
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			calledWith = arg
			res := reservedReservation()
			res.Status = dbgen.ReservationStatusClosedWa
			return res, nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	if err := svc.Close(context.Background(), seller.UUIDToString(testReservationID)); err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if calledWith.NewStatus != dbgen.ReservationStatusClosedWa {
		t.Errorf("expected new_status=closed-wa, got %s", calledWith.NewStatus)
	}
	if calledWith.ExpectedStatus != dbgen.ReservationStatusWaitingPay {
		t.Errorf("expected expected_status=waiting-pay, got %s", calledWith.ExpectedStatus)
	}
	if !calledWith.ClosedAt.Valid {
		t.Error("expected ClosedAt to be set")
	}
}

// ── Expire tests ──────────────────────────────────────────────────────────────

func TestExpire_FromReserved(t *testing.T) {
	var incrementCalled bool
	calls := 0

	db := &stubDB{
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			calls++
			if arg.ExpectedStatus == dbgen.ReservationStatusReserved {
				res := reservedReservation()
				res.Status = dbgen.ReservationStatusExpiredReleased
				return res, nil
			}
			return dbgen.Reservation{}, errNoRows
		},
		incrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			incrementCalled = true
			return okProduct(), nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	if err := svc.Expire(context.Background(), seller.UUIDToString(testReservationID)); err != nil {
		t.Fatalf("Expire error: %v", err)
	}
	if !incrementCalled {
		t.Error("expected IncrementStock to be called after expire from reserved")
	}
	if calls != 1 {
		t.Errorf("expected 1 UpdateReservationStatus call, got %d", calls)
	}
}

func TestExpire_FromWaitingPay(t *testing.T) {
	var incrementCalled bool
	calls := 0

	db := &stubDB{
		updateResStatus: func(_ context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			calls++
			switch arg.ExpectedStatus {
			case dbgen.ReservationStatusReserved:
				// Guard fires: not in reserved state.
				return dbgen.Reservation{}, errNoRows
			case dbgen.ReservationStatusWaitingPay:
				res := reservedReservation()
				res.Status = dbgen.ReservationStatusExpiredReleased
				return res, nil
			}
			return dbgen.Reservation{}, errNoRows
		},
		incrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			incrementCalled = true
			return okProduct(), nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	if err := svc.Expire(context.Background(), seller.UUIDToString(testReservationID)); err != nil {
		t.Fatalf("Expire error: %v", err)
	}
	if !incrementCalled {
		t.Error("expected IncrementStock to be called after expire from waiting-pay")
	}
	if calls != 2 {
		t.Errorf("expected 2 UpdateReservationStatus calls (reserved guard + waiting-pay), got %d", calls)
	}
}

func TestExpire_NoOpWhenAlreadyTerminal(t *testing.T) {
	// Both guards fire → already terminal (closed-wa or expired-released). No-op.
	var incrementCalled bool
	calls := 0

	db := &stubDB{
		updateResStatus: func(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
			calls++
			return dbgen.Reservation{}, errNoRows // guard fires for both reserved and waiting-pay
		},
		incrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			incrementCalled = true
			return okProduct(), nil
		},
	}

	svc := seller.NewReservationService(db, noopEnqueue)
	if err := svc.Expire(context.Background(), seller.UUIDToString(testReservationID)); err != nil {
		t.Fatalf("Expire no-op should not error, got: %v", err)
	}
	if incrementCalled {
		t.Error("IncrementStock must NOT be called when already terminal")
	}
	if calls != 2 {
		t.Errorf("expected 2 guard attempts, got %d", calls)
	}
}

// ── ParseUUID / UUIDToString round-trip ──────────────────────────────────────

func TestParseUUID_RoundTrip(t *testing.T) {
	original := "aaaabbbb-cccc-dddd-eeee-ffffffffffff"
	u, err := seller.ParseUUID(original)
	if err != nil {
		t.Fatalf("ParseUUID error: %v", err)
	}
	back := seller.UUIDToString(u)
	if back != original {
		t.Errorf("UUIDToString(ParseUUID(%q)) = %q, want original", original, back)
	}
}

func TestParseUUID_InvalidInput(t *testing.T) {
	cases := []string{"", "not-a-uuid", "gggggggg-0000-0000-0000-000000000000"}
	for _, s := range cases {
		if _, err := seller.ParseUUID(s); err == nil {
			t.Errorf("ParseUUID(%q) expected error, got nil", s)
		}
	}
}
