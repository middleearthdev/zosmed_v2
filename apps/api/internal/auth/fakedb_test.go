package auth

// fakedb_test.go — an in-memory dbgen.DBTX stub for the auth package's unit
// tests, following the same pattern as
// apps/api/internal/webhook/handler_test.go's fakeDedupeDTBX: a hand-rolled
// stub that dispatches on the (stable, sqlc-generated) SQL text, so Store/
// Handler/RequireUser can be exercised without a real Postgres connection.

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

type sessionRow struct {
	userID    pgtype.UUID
	expiresAt time.Time
}

// fakeAuthDBTX is an in-memory implementation of dbgen.DBTX covering the
// app_user / user_session / account queries the auth package uses.
type fakeAuthDBTX struct {
	mu           sync.Mutex
	usersByEmail map[string]dbgen.AppUser
	usersByID    map[pgtype.UUID]dbgen.AppUser
	sessions     map[string]sessionRow
	accounts     map[pgtype.UUID]dbgen.Account
	seq          int
}

func newFakeAuthDBTX() *fakeAuthDBTX {
	return &fakeAuthDBTX{
		usersByEmail: map[string]dbgen.AppUser{},
		usersByID:    map[pgtype.UUID]dbgen.AppUser{},
		sessions:     map[string]sessionRow{},
		accounts:     map[pgtype.UUID]dbgen.Account{},
	}
}

// newUUID mints a deterministic, distinct pgtype.UUID for in-memory rows.
func (f *fakeAuthDBTX) newUUID() pgtype.UUID {
	f.seq++
	var b [16]byte
	b[14] = byte(f.seq >> 8)
	b[15] = byte(f.seq)
	return pgtype.UUID{Bytes: b, Valid: true}
}

// putAccount seeds an account row directly (test setup helper — not a DBTX method).
func (f *fakeAuthDBTX) putAccount(userID pgtype.UUID, a dbgen.Account) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a.UserID = userID
	f.accounts[userID] = a
}

func (f *fakeAuthDBTX) Exec(_ context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case strings.Contains(sql, "INSERT INTO user_session"):
		tokenHash := args[0].(string)
		userID := args[1].(pgtype.UUID)
		expiresAt := args[2].(pgtype.Timestamptz)
		f.sessions[tokenHash] = sessionRow{userID: userID, expiresAt: expiresAt.Time}
		return pgconn.NewCommandTag("INSERT 0 1"), nil

	case strings.Contains(sql, "DELETE FROM user_session WHERE token_hash"):
		tokenHash := args[0].(string)
		delete(f.sessions, tokenHash)
		return pgconn.NewCommandTag("DELETE 1"), nil

	case strings.Contains(sql, "expires_at <= now()"):
		return pgconn.NewCommandTag("DELETE 0"), nil
	}
	panic("fakeAuthDBTX.Exec: unhandled query: " + sql)
}

func (f *fakeAuthDBTX) Query(_ context.Context, sql string, _ ...interface{}) (pgx.Rows, error) {
	panic("fakeAuthDBTX.Query: unexpected call for query: " + sql)
}

func (f *fakeAuthDBTX) QueryRow(_ context.Context, sql string, args ...interface{}) pgx.Row {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case strings.Contains(sql, "INSERT INTO app_user"):
		email := args[0].(string)
		passwordHash := args[1].(string)
		if _, exists := f.usersByEmail[email]; exists {
			return &errRow{err: &pgconn.PgError{Code: "23505"}}
		}
		u := dbgen.AppUser{
			ID:           f.newUUID(),
			Email:        email,
			PasswordHash: passwordHash,
			CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
		f.usersByEmail[email] = u
		f.usersByID[u.ID] = u
		return &userRow{user: u}

	case strings.Contains(sql, "FROM app_user WHERE email"):
		email := args[0].(string)
		u, ok := f.usersByEmail[email]
		if !ok {
			return &errRow{err: pgx.ErrNoRows}
		}
		return &userRow{user: u}

	case strings.Contains(sql, "FROM app_user WHERE id"):
		id := args[0].(pgtype.UUID)
		u, ok := f.usersByID[id]
		if !ok {
			return &errRow{err: pgx.ErrNoRows}
		}
		return &userRow{user: u}

	case strings.Contains(sql, "UPDATE app_user SET segment"):
		segment := args[0].(*string)
		id := args[1].(pgtype.UUID)
		u, ok := f.usersByID[id]
		if !ok {
			return &errRow{err: pgx.ErrNoRows}
		}
		u.Segment = segment
		f.usersByID[id] = u
		f.usersByEmail[u.Email] = u
		return &userRow{user: u}

	case strings.Contains(sql, "UPDATE app_user SET onboarding_completed_at"):
		id := args[0].(pgtype.UUID)
		u, ok := f.usersByID[id]
		if !ok {
			return &errRow{err: pgx.ErrNoRows}
		}
		u.OnboardingCompletedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		f.usersByID[id] = u
		f.usersByEmail[u.Email] = u
		return &userRow{user: u}

	case strings.Contains(sql, "JOIN app_user u ON u.id = s.user_id"):
		tokenHash := args[0].(string)
		sess, ok := f.sessions[tokenHash]
		if !ok || time.Now().After(sess.expiresAt) {
			return &errRow{err: pgx.ErrNoRows}
		}
		u, ok := f.usersByID[sess.userID]
		if !ok {
			return &errRow{err: pgx.ErrNoRows}
		}
		return &userRow{user: u}

	case strings.Contains(sql, "FROM account WHERE user_id"):
		id := args[0].(pgtype.UUID)
		a, ok := f.accounts[id]
		if !ok {
			return &errRow{err: pgx.ErrNoRows}
		}
		return &accountRow{account: a}
	}
	panic("fakeAuthDBTX.QueryRow: unhandled query: " + sql)
}

// ── row scanners ──────────────────────────────────────────────────────────────

// errRow implements pgx.Row, always failing Scan with a fixed error —
// used to model "no rows" / constraint-violation results.
type errRow struct{ err error }

func (r *errRow) Scan(_ ...interface{}) error { return r.err }

// userRow implements pgx.Row for the AppUser column order shared by every
// app_user query (CreateUser/GetUserByEmail/GetUserByID/SetUserSegment/
// CompleteOnboarding/GetSessionUser all RETURN/SELECT the same six columns).
type userRow struct{ user dbgen.AppUser }

func (r *userRow) Scan(dest ...interface{}) error {
	*(dest[0].(*pgtype.UUID)) = r.user.ID
	*(dest[1].(*string)) = r.user.Email
	*(dest[2].(*string)) = r.user.PasswordHash
	*(dest[3].(**string)) = r.user.Segment
	*(dest[4].(*pgtype.Timestamptz)) = r.user.OnboardingCompletedAt
	*(dest[5].(*pgtype.Timestamptz)) = r.user.CreatedAt
	return nil
}

// accountRow implements pgx.Row for the Account column order (matches
// libs/platform/dbgen/account.sql.go's GetAccountByUserID Scan order).
type accountRow struct{ account dbgen.Account }

func (r *accountRow) Scan(dest ...interface{}) error {
	*(dest[0].(*pgtype.UUID)) = r.account.ID
	*(dest[1].(*string)) = r.account.IgUserID
	*(dest[2].(*string)) = r.account.Handle
	*(dest[3].(*string)) = r.account.DisplayName
	*(dest[4].(*string)) = r.account.Status
	*(dest[5].(*pgtype.Timestamptz)) = r.account.CreatedAt
	*(dest[6].(*string)) = r.account.AccessToken
	*(dest[7].(*string)) = r.account.TokenType
	*(dest[8].(*[]string)) = r.account.Scopes
	*(dest[9].(*pgtype.Timestamptz)) = r.account.TokenExpiresAt
	*(dest[10].(*pgtype.Timestamptz)) = r.account.TokenRefreshedAt
	*(dest[11].(*pgtype.UUID)) = r.account.UserID
	return nil
}
