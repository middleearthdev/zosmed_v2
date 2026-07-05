package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	seller "github.com/zosmed/zosmed/libs/kits/seller"
)

// fakeLister is an in-memory expiredReservationLister.
type fakeLister struct {
	ids []pgtype.UUID
	err error
}

func (f *fakeLister) ListExpiredActiveReservations(_ context.Context, _ int32) ([]pgtype.UUID, error) {
	return f.ids, f.err
}

// fakeExpirer records the reservation IDs Expire was called with and can fail
// for specific IDs to prove one failure does not abort the sweep.
type fakeExpirer struct {
	called  []string
	failFor map[string]error
}

func (f *fakeExpirer) Expire(_ context.Context, reservationID string) error {
	f.called = append(f.called, reservationID)
	if err, ok := f.failFor[reservationID]; ok {
		return err
	}
	return nil
}

func uuidWithByte(b byte) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte{b}, Valid: true}
}

func TestReconcile_ExpiresAllDue(t *testing.T) {
	lister := &fakeLister{ids: []pgtype.UUID{uuidWithByte(1), uuidWithByte(2), uuidWithByte(3)}}
	expirer := &fakeExpirer{}
	h := NewReservationReconcileHandler(lister, expirer, silentLogger())

	if err := h.ProcessTask(context.Background(), nil); err != nil {
		t.Fatalf("ProcessTask error: %v", err)
	}
	if len(expirer.called) != 3 {
		t.Errorf("expected Expire called 3 times, got %d", len(expirer.called))
	}
}

func TestReconcile_OneFailureDoesNotAbortSweep(t *testing.T) {
	id1, id2, id3 := uuidWithByte(1), uuidWithByte(2), uuidWithByte(3)
	lister := &fakeLister{ids: []pgtype.UUID{id1, id2, id3}}
	// Middle reservation fails — the other two must still be attempted.
	expirer := &fakeExpirer{failFor: map[string]error{seller.UUIDToString(id2): errors.New("boom")}}
	h := NewReservationReconcileHandler(lister, expirer, silentLogger())

	if err := h.ProcessTask(context.Background(), nil); err != nil {
		t.Fatalf("ProcessTask should not return error on per-reservation failure, got: %v", err)
	}
	if len(expirer.called) != 3 {
		t.Errorf("expected all 3 reservations attempted despite one failure, got %d", len(expirer.called))
	}
}

func TestReconcile_ListErrorPropagates(t *testing.T) {
	lister := &fakeLister{err: errors.New("db down")}
	expirer := &fakeExpirer{}
	h := NewReservationReconcileHandler(lister, expirer, silentLogger())

	if err := h.ProcessTask(context.Background(), nil); err == nil {
		t.Fatal("expected error when list fails, got nil")
	}
	if len(expirer.called) != 0 {
		t.Errorf("Expire must not be called when list fails, got %d calls", len(expirer.called))
	}
}
