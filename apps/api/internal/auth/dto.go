package auth

import (
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// ValidSegments are the only accepted values for app_user.segment (mirrors the
// CHECK constraint in db/migrations/00008_app_user.sql — single source of
// truth for the enum lives in the DB; this is the app-side mirror needed to
// return a clean 400 before hitting Postgres). Segment selection is a user
// attribute (CLAUDE.md §8/§12a), not engine/kit logic — it deliberately does
// NOT import libs/kits/*.
var ValidSegments = map[string]bool{
	"seller":  true,
	"creator": true,
	"booking": true,
}

// UserDTO is the safe, external shape of app_user — NEVER include
// password_hash (ADR-003 AC-1/AC-8).
type UserDTO struct {
	ID                  string  `json:"id"`
	Email               string  `json:"email"`
	Segment             *string `json:"segment"`
	OnboardingCompleted bool    `json:"onboardingCompleted"`
}

// AccountDTO is the safe, external shape of the linked Instagram account —
// NEVER include access_token/token_* fields (ADR-003 AC-8).
type AccountDTO struct {
	Status      string `json:"status"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

// MeResponse is the payload for GET /api/v1/auth/me and the auth endpoints
// that also return account state (login).
type MeResponse struct {
	User    UserDTO     `json:"user"`
	Account *AccountDTO `json:"account"`
}

// RegisterRequest is the body of POST /api/v1/auth/register.
type RegisterRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Segment  *string `json:"segment,omitempty"`
}

// LoginRequest is the body of POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SegmentRequest is the body of PUT /api/v1/onboarding/segment.
type SegmentRequest struct {
	Segment string `json:"segment"`
}

// mapUserDTO maps a dbgen.AppUser to its safe external shape. Single mapping
// location (DRY §12a-1) — handlers never marshal dbgen.AppUser directly.
func mapUserDTO(u dbgen.AppUser) UserDTO {
	return UserDTO{
		ID:                  uuidx.Format(u.ID),
		Email:               u.Email,
		Segment:             u.Segment,
		OnboardingCompleted: u.OnboardingCompletedAt.Valid,
	}
}

// mapAccountDTO maps a dbgen.Account to its safe external shape.
func mapAccountDTO(a dbgen.Account) AccountDTO {
	return AccountDTO{
		Status:      a.Status,
		Handle:      a.Handle,
		DisplayName: a.DisplayName,
	}
}
