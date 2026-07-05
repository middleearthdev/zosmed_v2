package tasks

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// fakeTokenDTBX is a minimal dbgen.DBTX stub exercising refreshOne's two
// possible writes: UpdateAccountToken (success) and MarkAccountExpired
// (refresh failure). Both are :exec queries, so only Exec needs a fake body;
// Query/QueryRow are never reached from refreshOne and panic to surface
// accidental use.
type fakeTokenDTBX struct {
	execErr    error
	execCalls  []string // records which statement text each Exec call used (truncated), for assertions
	execTagRet pgconn.CommandTag
}

func (f *fakeTokenDTBX) Exec(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
	f.execCalls = append(f.execCalls, sql)
	return f.execTagRet, f.execErr
}

func (f *fakeTokenDTBX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	panic("fakeTokenDTBX.Query: unexpected call — refreshOne never lists accounts")
}

func (f *fakeTokenDTBX) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	panic("fakeTokenDTBX.QueryRow: unexpected call — refreshOne never lists accounts")
}

type fakeRefresher struct {
	err    error
	result igapi.LongLivedToken
}

func (f *fakeRefresher) RefreshLongLived(_ context.Context, _ string) (igapi.LongLivedToken, error) {
	if f.err != nil {
		return igapi.LongLivedToken{}, f.err
	}
	return f.result, nil
}

func testAccount() dbgen.Account {
	return dbgen.Account{
		ID:          pgtype.UUID{Bytes: [16]byte{0x01}, Valid: true},
		IgUserID:    "17841400",
		Handle:      "olshop_budi",
		Status:      "connected",
		AccessToken: "old-long-tok",
	}
}

func TestRefreshOne_Success_UpdatesToken(t *testing.T) {
	fakeDB := &fakeTokenDTBX{}
	refresher := &fakeRefresher{result: igapi.LongLivedToken{AccessToken: "new-long-tok", TokenType: "bearer", ExpiresIn: 5184000}}
	h := NewTokenRefreshHandler(dbgen.New(fakeDB), refresher, silentLogger())

	h.refreshOne(context.Background(), testAccount())

	if len(fakeDB.execCalls) != 1 {
		t.Fatalf("expected exactly 1 Exec call, got %d", len(fakeDB.execCalls))
	}
}

func TestRefreshOne_Failure_MarksExpired(t *testing.T) {
	fakeDB := &fakeTokenDTBX{}
	refresher := &fakeRefresher{err: errors.New("igapi: RefreshLongLived: dead token")}
	h := NewTokenRefreshHandler(dbgen.New(fakeDB), refresher, silentLogger())

	h.refreshOne(context.Background(), testAccount())

	if len(fakeDB.execCalls) != 1 {
		t.Fatalf("expected exactly 1 Exec call (mark expired), got %d", len(fakeDB.execCalls))
	}
}

func TestRefreshOne_UpdateFails_DoesNotPanic(t *testing.T) {
	fakeDB := &fakeTokenDTBX{execErr: errors.New("postgres: connection reset")}
	refresher := &fakeRefresher{result: igapi.LongLivedToken{AccessToken: "new-tok", ExpiresIn: 100}}
	h := NewTokenRefreshHandler(dbgen.New(fakeDB), refresher, silentLogger())

	// Must not panic even though the UpdateAccountToken Exec fails.
	h.refreshOne(context.Background(), testAccount())
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
