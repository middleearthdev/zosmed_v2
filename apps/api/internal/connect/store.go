package connect

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
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
// newly connected (or re-connected) account (ADR-002 §4.3).
type UpsertAccountParams struct {
	IgUserID       string
	Handle         string
	DisplayName    string
	AccessToken    string
	TokenType      string
	Scopes         []string
	TokenExpiresAt time.Time
}

// UpsertAccount inserts or updates the account row for the given IGSID,
// storing the long-lived token, its expiry, and granted scopes.
func (s *Store) UpsertAccount(ctx context.Context, p UpsertAccountParams) (dbgen.Account, error) {
	acc, err := s.q.UpsertAccountFromOAuth(ctx, dbgen.UpsertAccountFromOAuthParams{
		IgUserID:       p.IgUserID,
		Handle:         p.Handle,
		DisplayName:    p.DisplayName,
		AccessToken:    p.AccessToken,
		TokenType:      p.TokenType,
		Scopes:         p.Scopes,
		TokenExpiresAt: pgtype.Timestamptz{Time: p.TokenExpiresAt, Valid: true},
	})
	if err != nil {
		return dbgen.Account{}, fmt.Errorf("connect: upsert account: %w", err)
	}
	return acc, nil
}
