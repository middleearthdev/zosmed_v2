package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// refreshLeadWindow controls how far ahead of expiry accounts are swept up
// for refresh (ADR-002 §5 step 1). Long-lived tokens live ~60 days and must
// be at least 24h old to refresh (RESOLVED G9); a 7-day lead comfortably
// clears that minimum-age requirement on every sweep.
const refreshLeadWindow = 7 * 24 * time.Hour

// tokenRefresher is the subset of igapi.OAuthConfig that TokenRefreshHandler
// needs. Declared as an interface — like connect.oauthExchanger and
// workflow.Sender/Gater elsewhere in this codebase — purely so tests can
// inject a fake instead of hitting the network. igapi.OAuthConfig satisfies
// it structurally.
type tokenRefresher interface {
	RefreshLongLived(ctx context.Context, longToken string) (igapi.LongLivedToken, error)
}

// TokenRefreshHandler processes the periodic token:refresh-sweep task
// (ADR-002 §5). It refreshes long-lived IG user tokens that are close to
// expiry. Per-account failures are logged and mark that account 'expired' —
// they never fail the whole sweep (ADR-002 §5 step 2).
type TokenRefreshHandler struct {
	db    *dbgen.Queries
	oauth tokenRefresher
	log   *slog.Logger
}

// NewTokenRefreshHandler constructs a handler bound to the given DB and OAuth
// config. oauth is typically an igapi.OAuthConfig value, which satisfies
// tokenRefresher structurally; tests pass a fake instead.
func NewTokenRefreshHandler(db *dbgen.Queries, oauth tokenRefresher, log *slog.Logger) *TokenRefreshHandler {
	return &TokenRefreshHandler{db: db, oauth: oauth, log: log}
}

// ProcessTask implements the asynq handler signature. The task carries no
// payload — the sweep operates on whatever accounts are due, read fresh from
// Postgres each run.
func (h *TokenRefreshHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	threshold := pgtype.Timestamptz{Time: time.Now().Add(refreshLeadWindow), Valid: true}

	accounts, err := h.db.ListAccountsDueForRefresh(ctx, threshold)
	if err != nil {
		return fmt.Errorf("token_refresh: list accounts due for refresh: %w", err)
	}

	h.log.Info("token_refresh: sweep starting", slog.Int("accounts_due", len(accounts)))
	for _, acc := range accounts {
		h.refreshOne(ctx, acc)
	}
	return nil
}

// refreshOne refreshes a single account's token. Failures are logged and
// mark the account 'expired' rather than propagating — one bad account must
// never abort the sweep for the rest (ADR-002 §5 step 2).
func (h *TokenRefreshHandler) refreshOne(ctx context.Context, acc dbgen.Account) {
	log := h.log.With(slog.String("ig_user_id", acc.IgUserID))

	long, err := h.oauth.RefreshLongLived(ctx, acc.AccessToken)
	if err != nil {
		log.Warn("token_refresh: refresh failed — marking account expired", slog.String("error", err.Error()))
		if markErr := h.db.MarkAccountExpired(ctx, acc.ID); markErr != nil {
			log.Error("token_refresh: mark expired failed", slog.String("error", markErr.Error()))
		}
		return
	}

	expiresAt := time.Now().Add(time.Duration(long.ExpiresIn) * time.Second)
	err = h.db.UpdateAccountToken(ctx, dbgen.UpdateAccountTokenParams{
		ID:               acc.ID,
		AccessToken:      long.AccessToken,
		TokenExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
		TokenRefreshedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		log.Error("token_refresh: update token failed", slog.String("error", err.Error()))
		return
	}
	log.Info("token_refresh: refreshed", slog.Time("new_expires_at", expiresAt))
}
