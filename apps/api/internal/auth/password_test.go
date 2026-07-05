package auth

import "testing"

func TestHashPassword_VerifyPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("demo12345")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "demo12345" {
		t.Fatal("hash must not equal the plaintext password")
	}
	if !VerifyPassword(hash, "demo12345") {
		t.Error("expected VerifyPassword to accept the correct password")
	}
}

func TestVerifyPassword_WrongPassword_Fails(t *testing.T) {
	hash, err := HashPassword("demo12345")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Error("expected VerifyPassword to reject an incorrect password")
	}
}

func TestVerifyPassword_MalformedHash_Fails(t *testing.T) {
	if VerifyPassword("not-a-bcrypt-hash", "demo12345") {
		t.Error("expected VerifyPassword to reject a malformed hash")
	}
}

func TestHashPassword_DifferentSaltsPerCall(t *testing.T) {
	h1, err := HashPassword("demo12345")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := HashPassword("demo12345")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Error("expected two hashes of the same password to differ (bcrypt salts each call)")
	}
}
