package connect

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// Store persists OAuth connect results to Postgres via dbgen. handler.go
// never writes SQL directly — all access goes through this adapter (SoC §12a-3).
type Store struct {
	q *dbgen.Queries
}

// NewStore returns a Store backed by the given Queries.
func NewStore(q *dbgen.Queries) *Store {
	return &Store{q: q}
}

// UpsertAccountParams is the connect-flow-shaped input for persisting a
// newly connected (or re-connected) account (ADR-002 §4.3). UserID is the
// Zosmed user (ADR-003 §6) who initiated the connect flow, extracted from the
// verified signed state — empty when unknown (defensive; in practice Start
// now always requires a logged-in user, ADR-003 AC-9).
type UpsertAccountParams struct {
	IgUserID       string
	Handle         string
	DisplayName    string
	AccessToken    string
	TokenType      string
	Scopes         []string
	TokenExpiresAt time.Time
	UserID         string
}

// UpsertAccount inserts or updates the account row for the given IGSID,
// storing the long-lived token, its expiry, granted scopes, and the owning
// Zosmed user (if any).
func (s *Store) UpsertAccount(ctx context.Context, p UpsertAccountParams) (dbgen.Account, error) {
	var userID pgtype.UUID
	if p.UserID != "" {
		parsed, err := uuidx.Parse(p.UserID)
		if err != nil {
			return dbgen.Account{}, fmt.Errorf("connect: parse user id: %w", err)
		}
		userID = parsed
	}

	acc, err := s.q.UpsertAccountFromOAuth(ctx, dbgen.UpsertAccountFromOAuthParams{
		IgUserID:       p.IgUserID,
		Handle:         p.Handle,
		DisplayName:    p.DisplayName,
		AccessToken:    p.AccessToken,
		TokenType:      p.TokenType,
		Scopes:         p.Scopes,
		TokenExpiresAt: pgtype.Timestamptz{Time: p.TokenExpiresAt, Valid: true},
		UserID:         userID,
	})
	if err != nil {
		return dbgen.Account{}, fmt.Errorf("connect: upsert account: %w", err)
	}
	return acc, nil
}

// UserOnboardingComplete reports whether the Zosmed user identified by
// userID (a UUID string extracted from the verified connect-flow state) has
// finished onboarding (app_user.onboarding_completed_at is set). Used by
// Callback to choose the post-connect redirect target (ADR-003 §6). Returns
// false, nil for an empty/unparseable userID rather than erroring — Callback
// treats that as "not onboarded" (the safe default).
func (s *Store) UserOnboardingComplete(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	uid, err := uuidx.Parse(userID)
	if err != nil {
		return false, nil
	}
	u, err := s.q.GetUserByID(ctx, uid)
	if err != nil {
		return false, fmt.Errorf("connect: get user by id: %w", err)
	}
	return u.OnboardingCompletedAt.Valid, nil
}
