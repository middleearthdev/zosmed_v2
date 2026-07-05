package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// Sentinel errors returned by Store. Handlers translate these into the HTTP
// envelope (SoC §12a-3: store.go never writes an HTTP response).
var (
	// ErrEmailTaken is returned by CreateUser when the email is already registered.
	ErrEmailTaken = errors.New("auth: email already registered")
	// ErrNotFound is returned when a user/session/account lookup finds no row.
	ErrNotFound = errors.New("auth: not found")
)

// uniqueViolationCode is the Postgres error code for a unique constraint
// violation (23505), used to translate app_user.email UNIQUE into ErrEmailTaken.
const uniqueViolationCode = "23505"

// Store persists auth data to Postgres via dbgen. handler.go/onboarding.go/
// middleware.go never write SQL directly — everything routes through here
// (SoC §12a-3). A single concrete Store is used everywhere; no AuthService
// interface is introduced since there is only one implementation
// (anti-over-abstraction §12a-4, per ADR-003 §3).
type Store struct {
	q *dbgen.Queries
}

// NewStore returns a Store backed by the given Queries.
func NewStore(q *dbgen.Queries) *Store {
	return &Store{q: q}
}

// CreateUser inserts a new app_user with an already-hashed password.
// Returns ErrEmailTaken if the email is already registered.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (dbgen.AppUser, error) {
	u, err := s.q.CreateUser(ctx, dbgen.CreateUserParams{Email: email, PasswordHash: passwordHash})
	if err != nil {
		if isUniqueViolation(err) {
			return dbgen.AppUser{}, ErrEmailTaken
		}
		return dbgen.AppUser{}, fmt.Errorf("auth: create user: %w", err)
	}
	return u, nil
}

// UserByEmail looks up a user by (already-normalised, lower-case) email.
// Returns ErrNotFound if no such user exists.
func (s *Store) UserByEmail(ctx context.Context, email string) (dbgen.AppUser, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if isNoRows(err) {
			return dbgen.AppUser{}, ErrNotFound
		}
		return dbgen.AppUser{}, fmt.Errorf("auth: get user by email: %w", err)
	}
	return u, nil
}

// UserByID looks up a user by primary key. Returns ErrNotFound if absent.
func (s *Store) UserByID(ctx context.Context, id pgtype.UUID) (dbgen.AppUser, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return dbgen.AppUser{}, ErrNotFound
		}
		return dbgen.AppUser{}, fmt.Errorf("auth: get user by id: %w", err)
	}
	return u, nil
}

// CreateSession persists a new session row keyed by the SHA-256 hash of the
// raw token (the raw token itself never reaches Postgres — AC-2).
func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID pgtype.UUID, expiresAt time.Time) error {
	if err := s.q.CreateSession(ctx, dbgen.CreateSessionParams{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("auth: create session: %w", err)
	}
	return nil
}

// SessionUser resolves a session token hash to its owning user, rejecting
// expired sessions (the query itself filters expires_at > now()). Returns
// ErrNotFound for an absent/expired/revoked session — RequireUser treats
// that as "not logged in", never a 500.
func (s *Store) SessionUser(ctx context.Context, tokenHash string) (dbgen.AppUser, error) {
	u, err := s.q.GetSessionUser(ctx, tokenHash)
	if err != nil {
		if isNoRows(err) {
			return dbgen.AppUser{}, ErrNotFound
		}
		return dbgen.AppUser{}, fmt.Errorf("auth: get session user: %w", err)
	}
	return u, nil
}

// DeleteSession revokes a session (logout). Deleting an absent token is a no-op.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	if err := s.q.DeleteSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("auth: delete session: %w", err)
	}
	return nil
}

// SetSegment persists the chosen onboarding segment for a user.
func (s *Store) SetSegment(ctx context.Context, id pgtype.UUID, segment string) (dbgen.AppUser, error) {
	u, err := s.q.SetUserSegment(ctx, dbgen.SetUserSegmentParams{ID: id, Segment: &segment})
	if err != nil {
		return dbgen.AppUser{}, fmt.Errorf("auth: set user segment: %w", err)
	}
	return u, nil
}

// CompleteOnboarding stamps onboarding_completed_at = now() for a user.
// Callers must have already validated the completion preconditions
// (segment set + account connected) — this method has no side conditions.
func (s *Store) CompleteOnboarding(ctx context.Context, id pgtype.UUID) (dbgen.AppUser, error) {
	u, err := s.q.CompleteOnboarding(ctx, id)
	if err != nil {
		return dbgen.AppUser{}, fmt.Errorf("auth: complete onboarding: %w", err)
	}
	return u, nil
}

// AccountByUserID returns the one Instagram account linked to a user (MVP:
// one account per user, ADR-003 §2.2). Returns ErrNotFound if none is linked yet.
func (s *Store) AccountByUserID(ctx context.Context, userID pgtype.UUID) (dbgen.Account, error) {
	a, err := s.q.GetAccountByUserID(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return dbgen.Account{}, ErrNotFound
		}
		return dbgen.Account{}, fmt.Errorf("auth: get account by user id: %w", err)
	}
	return a, nil
}

// isNoRows reports whether err represents pgx's "no rows in result set".
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// isUniqueViolation reports whether err is a Postgres unique constraint
// violation (SQLSTATE 23505), used to translate the app_user.email UNIQUE
// constraint into the app-level ErrEmailTaken.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
