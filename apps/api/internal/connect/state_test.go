package connect

import (
	"strconv"
	"testing"
	"time"
)

func TestNewState_VerifyState_RoundTrip(t *testing.T) {
	state, err := NewState("app-secret", "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	userID, ok := VerifyState(state, "app-secret")
	if !ok {
		t.Error("expected freshly minted state to verify")
	}
	if userID != "user-123" {
		t.Errorf("expected userID=user-123, got %q", userID)
	}
}

func TestVerifyState_WrongSecret_Fails(t *testing.T) {
	state, err := NewState("app-secret", "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := VerifyState(state, "different-secret"); ok {
		t.Error("expected state signed with a different secret to fail verification")
	}
}

func TestVerifyState_Tampered_Fails(t *testing.T) {
	state, err := NewState("app-secret", "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tampered := state + "x"
	if _, ok := VerifyState(tampered, "app-secret"); ok {
		t.Error("expected tampered state to fail verification")
	}
}

func TestVerifyState_MalformedInput_Fails(t *testing.T) {
	cases := []string{"", "not-a-state", "a.b", "a.b.c.d.e"}
	for _, c := range cases {
		if _, ok := VerifyState(c, "app-secret"); ok {
			t.Errorf("expected malformed state %q to fail verification", c)
		}
	}
}

func TestVerifyState_Expired_Fails(t *testing.T) {
	// Build a state with a timestamp older than stateTTL, signed correctly.
	oldTs := time.Now().Add(-stateTTL - time.Minute).Unix()
	payload := "nonce123." + strconv.FormatInt(oldTs, 10) + ".user-123"
	sig := sign(payload, "app-secret")
	expired := payload + "." + sig

	if _, ok := VerifyState(expired, "app-secret"); ok {
		t.Error("expected expired state to fail verification")
	}
}

func TestVerifyState_EmptyUserID_Fails(t *testing.T) {
	state, err := NewState("app-secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := VerifyState(state, "app-secret"); ok {
		t.Error("expected state with empty embedded userID to fail verification")
	}
}
