package connect

import (
	"strconv"
	"testing"
	"time"
)

func TestNewState_VerifyState_RoundTrip(t *testing.T) {
	state, err := NewState("app-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !VerifyState(state, "app-secret") {
		t.Error("expected freshly minted state to verify")
	}
}

func TestVerifyState_WrongSecret_Fails(t *testing.T) {
	state, err := NewState("app-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if VerifyState(state, "different-secret") {
		t.Error("expected state signed with a different secret to fail verification")
	}
}

func TestVerifyState_Tampered_Fails(t *testing.T) {
	state, err := NewState("app-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tampered := state + "x"
	if VerifyState(tampered, "app-secret") {
		t.Error("expected tampered state to fail verification")
	}
}

func TestVerifyState_MalformedInput_Fails(t *testing.T) {
	cases := []string{"", "not-a-state", "a.b", "a.b.c.d"}
	for _, c := range cases {
		if VerifyState(c, "app-secret") {
			t.Errorf("expected malformed state %q to fail verification", c)
		}
	}
}

func TestVerifyState_Expired_Fails(t *testing.T) {
	// Build a state with a timestamp older than stateTTL, signed correctly.
	oldTs := time.Now().Add(-stateTTL - time.Minute).Unix()
	payload := "nonce123." + strconv.FormatInt(oldTs, 10)
	sig := sign(payload, "app-secret")
	expired := payload + "." + sig

	if VerifyState(expired, "app-secret") {
		t.Error("expected expired state to fail verification")
	}
}
