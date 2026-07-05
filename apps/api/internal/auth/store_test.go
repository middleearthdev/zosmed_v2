package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

func newTestStore() (*Store, *fakeAuthDBTX) {
	db := newFakeAuthDBTX()
	return NewStore(dbgen.New(db)), db
}

func TestStore_CreateUser_Success(t *testing.T) {
	store, _ := newTestStore()
	u, err := store.CreateUser(context.Background(), "buyer@example.com", "hashed")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.Email != "buyer@example.com" {
		t.Errorf("expected email to round-trip, got %q", u.Email)
	}
	if u.PasswordHash != "hashed" {
		t.Errorf("expected password hash to round-trip, got %q", u.PasswordHash)
	}
}

func TestStore_CreateUser_DuplicateEmail_ReturnsErrEmailTaken(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	if _, err := store.CreateUser(ctx, "dup@example.com", "hash1"); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	_, err := store.CreateUser(ctx, "dup@example.com", "hash2")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestStore_UserByEmail_NotFound(t *testing.T) {
	store, _ := newTestStore()
	_, err := store.UserByEmail(context.Background(), "ghost@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_CreateSession_SessionUser_RoundTrip(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "session@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := store.CreateSession(ctx, "hash-of-token", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := store.SessionUser(ctx, "hash-of-token")
	if err != nil {
		t.Fatalf("SessionUser: %v", err)
	}
	if got.Email != u.Email {
		t.Errorf("expected SessionUser to resolve to the same user, got email %q", got.Email)
	}
}

func TestStore_SessionUser_Expired_ReturnsErrNotFound(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "expired@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := store.CreateSession(ctx, "expired-token-hash", u.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	_, err = store.SessionUser(ctx, "expired-token-hash")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for an expired session, got %v", err)
	}
}

func TestStore_DeleteSession_Revokes(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "logout@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := store.CreateSession(ctx, "logout-token-hash", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := store.DeleteSession(ctx, "logout-token-hash"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	_, err = store.SessionUser(ctx, "logout-token-hash")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after DeleteSession, got %v", err)
	}
}

func TestStore_SetSegment_CompleteOnboarding(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "onboard@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.OnboardingCompletedAt.Valid {
		t.Fatal("expected a freshly created user to not have onboarding completed")
	}

	updated, err := store.SetSegment(ctx, u.ID, "seller")
	if err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	if updated.Segment == nil || *updated.Segment != "seller" {
		t.Errorf("expected segment=seller, got %v", updated.Segment)
	}

	completed, err := store.CompleteOnboarding(ctx, u.ID)
	if err != nil {
		t.Fatalf("CompleteOnboarding: %v", err)
	}
	if !completed.OnboardingCompletedAt.Valid {
		t.Error("expected onboarding_completed_at to be set")
	}
}

func TestStore_AccountByUserID_NotFound(t *testing.T) {
	store, _ := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "noaccount@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	_, err = store.AccountByUserID(ctx, u.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_AccountByUserID_Found(t *testing.T) {
	store, db := newTestStore()
	ctx := context.Background()
	u, err := store.CreateUser(ctx, "hasaccount@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	db.putAccount(u.ID, dbgen.Account{
		IgUserID:    "17841400",
		Handle:      "olshop.aurora",
		DisplayName: "Aurora Olshop",
		Status:      "connected",
	})

	acc, err := store.AccountByUserID(ctx, u.ID)
	if err != nil {
		t.Fatalf("AccountByUserID: %v", err)
	}
	if acc.Handle != "olshop.aurora" {
		t.Errorf("expected handle to round-trip, got %q", acc.Handle)
	}
}
