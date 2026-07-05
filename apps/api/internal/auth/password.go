package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the bcrypt work factor (ADR-003 §0 decision 1 / AC-4).
// This is the ONLY place password hashing happens in the codebase (DRY §12a-1).
const bcryptCost = 12

// MaxPasswordLen is bcrypt's hard input limit: golang.org/x/crypto rejects
// passwords longer than 72 bytes with ErrPasswordTooLong (it does NOT silently
// truncate), so callers must reject >72 with a clean 400 before hashing.
const MaxPasswordLen = 72

// dummyPasswordHash is a valid bcrypt hash of a random string, computed once at
// startup. Login runs VerifyPassword against it when the email is unknown so the
// response time matches the "wrong password" path — closing the user-enumeration
// timing side-channel (review MAJOR-1). Its plaintext is unknowable, so it can
// never match a real password.
var dummyPasswordHash = mustHashRandom()

func mustHashRandom() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic("auth: cannot seed dummy password hash: " + err.Error())
	}
	h, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(b)), bcryptCost)
	if err != nil {
		panic("auth: cannot compute dummy password hash: " + err.Error())
	}
	return string(h)
}

// VerifyDummyPassword burns one bcrypt comparison against dummyPasswordHash to
// equalise login timing when the account does not exist. Result is discarded.
func VerifyDummyPassword(password string) {
	_ = VerifyPassword(dummyPasswordHash, password)
}

// HashPassword bcrypt-hashes a plaintext password. The plaintext is never
// logged or persisted anywhere else in the codebase (AC-4).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword reports whether password matches the given bcrypt hash.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
