package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// seedUser creates a user via the store and returns both its dbgen row and
// the UserDTO shape RequireUser would have injected into context.
func seedUser(t *testing.T, store *Store, email string) (dbgen.AppUser, UserDTO) {
	t.Helper()
	u, err := store.CreateUser(context.Background(), email, "hash")
	if err != nil {
		t.Fatalf("seedUser CreateUser: %v", err)
	}
	return u, mapUserDTO(u)
}

func withUserCtx(req *http.Request, dto UserDTO) *http.Request {
	return req.WithContext(WithUser(req.Context(), dto))
}

// ── PUT /api/v1/onboarding/segment ───────────────────────────────────────────

func TestPutSegment_ValidSegment_Persists(t *testing.T) {
	h, store, _ := newTestHandler()
	_, dto := seedUser(t, store, "segment@example.com")

	body := `{"segment":"seller"}`
	req := withUserCtx(httptest.NewRequest(http.MethodPut, "/api/v1/onboarding/segment", bytes.NewBufferString(body)), dto)
	w := httptest.NewRecorder()
	h.PutSegment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	var got MeResponse
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if got.User.Segment == nil || *got.User.Segment != "seller" {
		t.Errorf("expected segment=seller, got %v", got.User.Segment)
	}
}

func TestPutSegment_InvalidSegment_Returns400(t *testing.T) {
	h, store, _ := newTestHandler()
	_, dto := seedUser(t, store, "badsegment@example.com")

	body := `{"segment":"reseller"}`
	req := withUserCtx(httptest.NewRequest(http.MethodPut, "/api/v1/onboarding/segment", bytes.NewBufferString(body)), dto)
	w := httptest.NewRecorder()
	h.PutSegment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPutSegment_NoUserInContext_Returns401(t *testing.T) {
	h, _, _ := newTestHandler()
	body := `{"segment":"seller"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/onboarding/segment", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.PutSegment(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── POST /api/v1/onboarding/complete ─────────────────────────────────────────

func TestCompleteOnboarding_SegmentMissing_Returns409(t *testing.T) {
	h, store, _ := newTestHandler()
	_, dto := seedUser(t, store, "nosegment@example.com")

	req := withUserCtx(httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", nil), dto)
	w := httptest.NewRecorder()
	h.CompleteOnboarding(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "onboarding_incomplete" || env.Error.Reason != "segment_missing" {
		t.Errorf("expected onboarding_incomplete/segment_missing, got %+v", env.Error)
	}
}

func TestCompleteOnboarding_AccountNotConnected_Returns409(t *testing.T) {
	h, store, _ := newTestHandler()
	u, _ := seedUser(t, store, "noaccount2@example.com")
	updated, err := store.SetSegment(context.Background(), u.ID, "seller")
	if err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	dto := mapUserDTO(updated)

	req := withUserCtx(httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", nil), dto)
	w := httptest.NewRecorder()
	h.CompleteOnboarding(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Error == nil || env.Error.Code != "onboarding_incomplete" || env.Error.Reason != "account_not_connected" {
		t.Errorf("expected onboarding_incomplete/account_not_connected, got %+v", env.Error)
	}
}

func TestCompleteOnboarding_AccountNotYetConnected_StatusOtherThanConnected_Returns409(t *testing.T) {
	h, store, db := newTestHandler()
	u, _ := seedUser(t, store, "expiredaccount@example.com")
	updated, err := store.SetSegment(context.Background(), u.ID, "seller")
	if err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	db.putAccount(u.ID, dbgen.Account{IgUserID: "1", Handle: "h", Status: "expired"})
	dto := mapUserDTO(updated)

	req := withUserCtx(httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", nil), dto)
	w := httptest.NewRecorder()
	h.CompleteOnboarding(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a non-connected (expired) account, got %d", w.Code)
	}
}

func TestCompleteOnboarding_AllPreconditionsMet_Succeeds(t *testing.T) {
	h, store, db := newTestHandler()
	u, _ := seedUser(t, store, "complete@example.com")
	updated, err := store.SetSegment(context.Background(), u.ID, "seller")
	if err != nil {
		t.Fatalf("SetSegment: %v", err)
	}
	db.putAccount(u.ID, dbgen.Account{IgUserID: "1", Handle: "olshop.aurora", Status: "connected"})
	dto := mapUserDTO(updated)

	req := withUserCtx(httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", nil), dto)
	w := httptest.NewRecorder()
	h.CompleteOnboarding(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	var got MeResponse
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if !got.User.OnboardingCompleted {
		t.Error("expected onboardingCompleted=true after CompleteOnboarding")
	}

	// Verify the store was actually stamped, not just the response DTO.
	fresh, err := store.UserByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	if !fresh.OnboardingCompletedAt.Valid {
		t.Error("expected onboarding_completed_at to be persisted")
	}
}

func TestCompleteOnboarding_NoUserInContext_Returns401(t *testing.T) {
	h, _, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", nil)
	w := httptest.NewRecorder()
	h.CompleteOnboarding(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
