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
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// ── In-memory test double ─────────────────────────────────────────────────────

// errNoRows mimics pgx.ErrNoRows via the string suffix that isNoRows checks.
var errNoRows = errors.New("no rows in result set")

func mustUUID(s string) pgtype.UUID {
	u, err := uuidx.Parse(s)
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
	getProduct              func(ctx context.Context, arg dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error)
	decrementStock          func(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	incrementStock          func(ctx context.Context, id pgtype.UUID) (dbgen.Product, error)
	createReservation       func(ctx context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error)
	getReservationByComment func(ctx context.Context, arg dbgen.GetReservationByCommentParams) (dbgen.Reservation, error)
	updateResStatus         func(ctx context.Context, arg dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error)
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
func (s *stubDB) GetReservationByComment(ctx context.Context, arg dbgen.GetReservationByCommentParams) (dbgen.Reservation, error) {
	return s.getReservationByComment(ctx, arg)
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
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
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
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
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

// TestReserve_CreateFails_NoEnqueue verifies MAJOR-3a: when CreateReservation
// fails, Reserve returns the error and does NOT enqueue an expire timer. In
// production DecrementStock + CreateReservation share one transaction, so the
// stock decrement is rolled back too (atomicity guaranteed by NewPgxTxRunner;
// the in-memory double models only the error propagation + no-enqueue).
func TestReserve_CreateFails_NoEnqueue(t *testing.T) {
	errCreate := errors.New("insert failed: connection reset")
	enqueued := false

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) { return okProduct(), nil },
		createReservation: func(_ context.Context, _ dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			return dbgen.Reservation{}, errCreate
		},
	}

	svc := seller.NewReservationService(db, func(_ context.Context, _ string, _ time.Duration) error {
		enqueued = true
		return nil
	})

	_, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-fail", "user-f", "Fajar", "6281234567890", 0)

	if err == nil {
		t.Fatal("expected error when CreateReservation fails, got nil")
	}
	if !errors.Is(err, errCreate) {
		t.Errorf("expected wrapped create error, got %v", err)
	}
	if enqueued {
		t.Error("expire timer must NOT be enqueued when reservation creation fails")
	}
}

// recordingTxRunner is a TxRunner that models transaction outcome: it commits
// when fn succeeds and rolls back when fn returns an error, recording which
// happened. Lets us assert MAJOR-3a atomicity at the seam without a real DB.
type recordingTxRunner struct {
	db         seller.ReservationDB
	committed  bool
	rolledBack bool
}

func (r *recordingTxRunner) InTx(ctx context.Context, fn func(q seller.ReservationDB) error) error {
	if err := fn(r.db); err != nil {
		r.rolledBack = true
		return err
	}
	r.committed = true
	return nil
}

// TestReserve_CreateFails_RollsBack verifies MAJOR-3a atomicity: when
// CreateReservation fails inside the transaction, the runner rolls back (so the
// preceding DecrementStock is undone) and never commits.
func TestReserve_CreateFails_RollsBack(t *testing.T) {
	decremented := false
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			decremented = true
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, _ dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			return dbgen.Reservation{}, errors.New("insert failed")
		},
	}
	runner := &recordingTxRunner{db: db}
	svc := seller.NewReservationServiceTx(db, runner, noopEnqueue)

	_, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-rb", "user-rb", "Rina", "6281234567890", 0)

	if err == nil {
		t.Fatal("expected error when CreateReservation fails")
	}
	if !decremented {
		t.Error("DecrementStock should have run inside the transaction")
	}
	if !runner.rolledBack {
		t.Error("transaction must roll back when CreateReservation fails (stock not leaked)")
	}
	if runner.committed {
		t.Error("transaction must NOT commit on failure")
	}
}

// TestReserve_Success_Commits verifies the happy path commits the transaction.
func TestReserve_Success_Commits(t *testing.T) {
	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) { return okProduct(), nil },
		createReservation: func(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			res := reservedReservation()
			res.WaLink = arg.WaLink
			return res, nil
		},
	}
	runner := &recordingTxRunner{db: db}
	svc := seller.NewReservationServiceTx(db, runner, noopEnqueue)

	if _, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-ok", "user-ok", "Oki", "6281234567890", 0); err != nil {
		t.Fatalf("Reserve error: %v", err)
	}
	if !runner.committed {
		t.Error("transaction must commit on success")
	}
	if runner.rolledBack {
		t.Error("transaction must NOT roll back on success")
	}
}

// TestReserve_Idempotent_ConflictRollsBackAndReturnsExisting verifies
// ADR-007 §2.3b (#6b): when CreateReservation reports the ON CONFLICT
// DO NOTHING branch (isNoRows) for a comment that was already reserved by an
// earlier attempt, Reserve must:
//   - roll back the tx (so the DecrementStock above is undone — no double
//     decrement),
//   - return the EXISTING reservation (via GetReservationByComment) with a
//     nil error (idempotent success, not a hard failure),
//   - NOT enqueue a second expire timer.
func TestReserve_Idempotent_ConflictRollsBackAndReturnsExisting(t *testing.T) {
	existing := reservedReservation()
	decremented := 0
	lookedUpWith := dbgen.GetReservationByCommentParams{}

	db := &stubDB{
		getProduct: func(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
			return okProduct(), nil
		},
		decrementStock: func(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
			decremented++
			return okProduct(), nil
		},
		createReservation: func(_ context.Context, _ dbgen.CreateReservationParams) (dbgen.Reservation, error) {
			// Simulates the UNIQUE(account_id, ig_comment_id) ON CONFLICT DO
			// NOTHING branch (db/query/reservations.sql) surfacing as zero rows.
			return dbgen.Reservation{}, errNoRows
		},
		getReservationByComment: func(_ context.Context, arg dbgen.GetReservationByCommentParams) (dbgen.Reservation, error) {
			lookedUpWith = arg
			return existing, nil
		},
	}
	runner := &recordingTxRunner{db: db}
	enqueued := 0
	svc := seller.NewReservationServiceTx(db, runner, func(_ context.Context, _ string, _ time.Duration) error {
		enqueued++
		return nil
	})

	result, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-dup", "user-dup", "Dupa", "6281234567890", 0)

	if err != nil {
		t.Fatalf("Reserve should be idempotent (no error) on conflict, got: %v", err)
	}
	if result.Reservation.ID != existing.ID {
		t.Errorf("expected the existing reservation to be returned, got %+v", result.Reservation)
	}
	if result.ProductName != "Kaos Ungu M" {
		t.Errorf("expected ProductName still populated from the product lookup, got %q", result.ProductName)
	}
	if lookedUpWith.IgCommentID != "comment-dup" {
		t.Errorf("GetReservationByComment called with ig_comment_id=%q, want comment-dup", lookedUpWith.IgCommentID)
	}
	if !runner.rolledBack {
		t.Error("tx must roll back on CreateReservation conflict (stock not double-decremented)")
	}
	if runner.committed {
		t.Error("tx must NOT commit on conflict")
	}
	if decremented != 1 {
		t.Errorf("DecrementStock should be attempted once inside the (rolled-back) tx, got %d calls", decremented)
	}
	if enqueued != 0 {
		t.Error("expire timer must NOT be re-enqueued for an already-reserved comment")
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
	resIDStr := uuidx.Format(testReservationID)

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
	if err := svc.MarkWaitingPay(context.Background(), uuidx.Format(testReservationID)); err != nil {
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
	if err := svc.Close(context.Background(), uuidx.Format(testReservationID)); err != nil {
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
	if err := svc.Expire(context.Background(), uuidx.Format(testReservationID)); err != nil {
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
	if err := svc.Expire(context.Background(), uuidx.Format(testReservationID)); err != nil {
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
	if err := svc.Expire(context.Background(), uuidx.Format(testReservationID)); err != nil {
		t.Fatalf("Expire no-op should not error, got: %v", err)
	}
	if incrementCalled {
		t.Error("IncrementStock must NOT be called when already terminal")
	}
	if calls != 2 {
		t.Errorf("expected 2 guard attempts, got %d", calls)
	}
}

// ── ADR-007 §5 scenario 11: ingest-rerun-level Reserve idempotency ───────────
//
// TestReserve_Idempotent_ConflictRollsBackAndReturnsExisting (above) proves
// the conflict BRANCH in isolation by hand-crafting CreateReservation's
// ON-CONFLICT response. The test below instead calls svc.Reserve TWICE with
// the identical (account, comment) pair — modelling an actual asynq re-run of
// comment:ingest end-to-end — against a stateful fake DB (no real Postgres,
// per the ADR-007 §5 scenario 11 note: "tambahkan level ingest-rerun bila
// feasible tanpa Postgres nyata"). rollbackAwareTxRunner undoes the stock
// decrement on any fn error, the same externally-visible effect a real
// Postgres transaction rollback gives NewPgxTxRunner in production — so this
// is the smallest fake that can honestly assert "stock decremented once net
// across both calls", not just "DecrementStock was attempted once".

// statefulReservationDB is a stateful (not scripted) ReservationDB double:
// stock and reservations actually mutate across calls, so two svc.Reserve
// calls in a row behave like two real database round-trips.
type statefulReservationDB struct {
	product      dbgen.Product
	stockLeft    int32
	reservations map[string]dbgen.Reservation // keyed by ig_comment_id

	decrementCalls int
	createCalls    int
	nextSeq        int
}

func (d *statefulReservationDB) GetProductByPostAndCode(_ context.Context, _ dbgen.GetProductByPostAndCodeParams) (dbgen.Product, error) {
	p := d.product
	p.StockLeft = d.stockLeft
	return p, nil
}

func (d *statefulReservationDB) DecrementStock(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
	d.decrementCalls++
	if d.stockLeft <= 0 {
		return dbgen.Product{}, errNoRows
	}
	d.stockLeft--
	p := d.product
	p.StockLeft = d.stockLeft
	return p, nil
}

func (d *statefulReservationDB) IncrementStock(_ context.Context, _ pgtype.UUID) (dbgen.Product, error) {
	d.stockLeft++
	p := d.product
	p.StockLeft = d.stockLeft
	return p, nil
}

func (d *statefulReservationDB) CreateReservation(_ context.Context, arg dbgen.CreateReservationParams) (dbgen.Reservation, error) {
	d.createCalls++
	if _, exists := d.reservations[arg.IgCommentID]; exists {
		// Models the real UNIQUE(account_id, ig_comment_id) ON CONFLICT DO
		// NOTHING branch (db/query/reservations.sql) surfacing as zero rows.
		return dbgen.Reservation{}, errNoRows
	}
	d.nextSeq++
	res := dbgen.Reservation{
		ID:            mustUUID(fmt.Sprintf("eeeeeeee-0000-0000-0000-%012d", d.nextSeq)),
		AccountID:     arg.AccountID,
		CatalogPostID: arg.CatalogPostID,
		ProductID:     arg.ProductID,
		Code:          arg.Code,
		Status:        dbgen.ReservationStatusReserved,
		HoldSeconds:   arg.HoldSeconds,
		ExpiresAt:     arg.ExpiresAt,
		WaLink:        arg.WaLink,
	}
	d.reservations[arg.IgCommentID] = res
	return res, nil
}

func (d *statefulReservationDB) GetReservationByComment(_ context.Context, arg dbgen.GetReservationByCommentParams) (dbgen.Reservation, error) {
	res, ok := d.reservations[arg.IgCommentID]
	if !ok {
		return dbgen.Reservation{}, errNoRows
	}
	return res, nil
}

func (d *statefulReservationDB) UpdateReservationStatus(_ context.Context, _ dbgen.UpdateReservationStatusParams) (dbgen.Reservation, error) {
	return dbgen.Reservation{}, errNoRows
}

// rollbackAwareTxRunner undoes the stock decrement when fn returns an error —
// the same externally-observable effect a real Postgres ROLLBACK has on
// NewPgxTxRunner in production (MAJOR-3a).
type rollbackAwareTxRunner struct{ db *statefulReservationDB }

func (r rollbackAwareTxRunner) InTx(_ context.Context, fn func(q seller.ReservationDB) error) error {
	stockBefore := r.db.stockLeft
	if err := fn(r.db); err != nil {
		r.db.stockLeft = stockBefore
		return err
	}
	return nil
}

// TestReserve_RerunSameComment_IngestLevel_StockDecrementedOnce is ADR-007
// §5 scenario 11 at the ingest-rerun level: calling svc.Reserve TWICE for the
// SAME (account, comment) — as an asynq retry of comment:ingest would —
// must produce exactly ONE reservation, decrement stock exactly ONCE net,
// enqueue exactly ONE expire timer, and return the SAME reservation both
// times (idempotent success, not a hard failure on the second call).
func TestReserve_RerunSameComment_IngestLevel_StockDecrementedOnce(t *testing.T) {
	seed := okProduct()
	db := &statefulReservationDB{
		product:      seed,
		stockLeft:    seed.StockLeft,
		reservations: map[string]dbgen.Reservation{},
	}
	runner := rollbackAwareTxRunner{db: db}
	enqueueCalls := 0
	svc := seller.NewReservationServiceTx(db, runner, func(_ context.Context, _ string, _ time.Duration) error {
		enqueueCalls++
		return nil
	})

	result1, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-rerun-1", "user-1", "Rina", "6281234567890", 0)
	if err != nil {
		t.Fatalf("first Reserve (initial ingest run): %v", err)
	}

	// Simulated asynq re-run of the SAME comment:ingest task (identical
	// account+comment) — e.g. a crash/retry after the first Reserve already
	// committed.
	result2, err := svc.Reserve(context.Background(),
		testAccountID, testCatalogPostID, "C1",
		"comment-rerun-1", "user-1", "Rina", "6281234567890", 0)
	if err != nil {
		t.Fatalf("second (rerun) Reserve: expected idempotent success, got error: %v", err)
	}

	if result1.Reservation.ID != result2.Reservation.ID {
		t.Errorf("expected the SAME reservation on rerun, got %v vs %v", result1.Reservation.ID, result2.Reservation.ID)
	}
	if db.createCalls != 2 {
		t.Errorf("expected 2 CreateReservation attempts (1 insert + 1 conflict), got %d", db.createCalls)
	}
	wantStock := seed.StockLeft - 1
	if db.stockLeft != wantStock {
		t.Errorf("expected stock decremented exactly ONCE net across both Reserve calls: stockLeft=%d, want %d", db.stockLeft, wantStock)
	}
	if enqueueCalls != 1 {
		t.Errorf("expected exactly 1 expire timer enqueued (not re-enqueued on rerun), got %d", enqueueCalls)
	}
}

// UUID parse/format round-trip is covered by libs/platform/uuidx (M5).
