// Package connect implements Business Login for Instagram (Instagram Login,
// CLAUDE.md §4.0) for Zosmed: GET /connect/instagram redirects the user to
// Instagram's consent screen, and GET /connect/instagram/callback exchanges
// the returned code for a long-lived IG-user token and persists the account
// (ADR-002 §3).
package connect

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/zosmed/zosmed/apps/api/internal/auth"
	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// oauthExchanger is the subset of igapi.OAuthConfig that Callback needs.
// Declared as an interface — like workflow.Sender/workflow.Gater elsewhere in
// this codebase — purely so tests can inject a fake instead of hitting the
// network. igapi.OAuthConfig satisfies it structurally.
type oauthExchanger interface {
	ExchangeCode(ctx context.Context, code string) (igapi.ShortLivedToken, error)
	ExchangeLongLived(ctx context.Context, shortToken string) (igapi.LongLivedToken, error)
}

// identityResolver is the subset of igapi.Client that Callback needs, to
// resolve the IGSID (account.ig_user_id) once a long-lived token is minted.
type identityResolver interface {
	Me(ctx context.Context) (igapi.MeResult, error)
}

// accountUpserter is the persistence seam Handler needs; Store satisfies it.
type accountUpserter interface {
	UpsertAccount(ctx context.Context, p UpsertAccountParams) (dbgen.Account, error)
	// UserOnboardingComplete reports whether userID has finished onboarding
	// (ADR-003 §6), used to pick the post-connect redirect target.
	UserOnboardingComplete(ctx context.Context, userID string) (bool, error)
}

// Handler implements the connect (OAuth) HTTP endpoints.
type Handler struct {
	oauth     oauthExchanger
	authorize func(state string, scopes []string) string
	newClient func(accessToken string) identityResolver
	appSecret string // signs the anti-CSRF state; same App Secret as OAuth client_secret (DRY §12a-1)
	store     accountUpserter
	log       *slog.Logger
}

// New wires a Handler for production use: real Instagram OAuth + a Postgres-backed Store.
func New(oauth igapi.OAuthConfig, appSecret string, store *Store, log *slog.Logger) *Handler {
	return &Handler{
		oauth:     oauth,
		authorize: oauth.AuthorizeURL,
		newClient: func(accessToken string) identityResolver { return igapi.New(accessToken) },
		appSecret: appSecret,
		store:     store,
		log:       log,
	}
}

// Start handles GET /connect/instagram: redirects to Instagram's authorize
// screen with a signed anti-CSRF state (ADR-002 §3.1 step 1), embedding the
// logged-in Zosmed user id (ADR-003 AC-9) so Callback can link the resulting
// account. Mounted behind RequireUser (httpx/router.go) — a user is always
// present in context here.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Silakan login terlebih dahulu")
		return
	}

	state, err := NewState(h.appSecret, user.ID)
	if err != nil {
		h.log.Error("connect: generate state", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "state_generation_failed", "Gagal memulai koneksi Instagram, coba lagi")
		return
	}
	http.Redirect(w, r, h.authorize(state, igapi.DefaultScopes), http.StatusFound)
}

// Callback handles GET /connect/instagram/callback: verifies state, exchanges
// the code for a long-lived token, resolves the IGSID, and persists the
// account (ADR-002 §3.1 steps 2-6).
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	if igErr := q.Get("error"); igErr != "" {
		h.log.Warn("connect: user denied or Instagram returned an error", slog.String("ig_error", igErr))
		httpx.Err(w, http.StatusBadRequest, "oauth_denied", "Koneksi Instagram dibatalkan")
		return
	}

	userID, ok := VerifyState(q.Get("state"), h.appSecret)
	if !ok {
		h.log.Warn("connect: invalid or expired state")
		httpx.Err(w, http.StatusForbidden, "invalid_state", "Sesi koneksi tidak valid atau kedaluwarsa, coba lagi")
		return
	}

	code := q.Get("code")
	if code == "" {
		httpx.Err(w, http.StatusBadRequest, "missing_code", "Kode otorisasi Instagram tidak ditemukan")
		return
	}

	short, err := h.oauth.ExchangeCode(ctx, code)
	if err != nil {
		h.log.Error("connect: exchange code", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusBadGateway, "exchange_code_failed", "Gagal menghubungkan akun Instagram")
		return
	}

	long, err := h.oauth.ExchangeLongLived(ctx, short.AccessToken)
	if err != nil {
		h.log.Error("connect: exchange long-lived token", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusBadGateway, "exchange_long_lived_failed", "Gagal memperpanjang token Instagram")
		return
	}

	me, err := h.newClient(long.AccessToken).Me(ctx)
	if err != nil {
		h.log.Error("connect: fetch identity", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusBadGateway, "fetch_identity_failed", "Gagal mengambil identitas akun Instagram")
		return
	}

	tokenType := long.TokenType
	if tokenType == "" {
		tokenType = "bearer"
	}
	expiresAt := time.Now().Add(time.Duration(long.ExpiresIn) * time.Second)

	if _, err := h.store.UpsertAccount(ctx, UpsertAccountParams{
		IgUserID:       me.UserID,
		Handle:         me.Username,
		DisplayName:    me.Username,
		AccessToken:    long.AccessToken,
		TokenType:      tokenType,
		Scopes:         igapi.DefaultScopes,
		TokenExpiresAt: expiresAt,
		UserID:         userID,
	}); err != nil {
		h.log.Error("connect: persist account", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "persist_account_failed", "Gagal menyimpan akun Instagram")
		return
	}

	h.log.Info("connect: account connected", slog.String("ig_user_id", me.UserID), slog.String("handle", me.Username))

	// Redirect target depends on onboarding state (ADR-003 §0 decision 3):
	// still onboarding -> back to /onboarding to continue the flow; already
	// onboarded (re-connecting from Settings) -> /settings. Default safe on
	// lookup failure is /onboarding.
	redirectTo := "/onboarding?connected=1"
	if completed, err := h.store.UserOnboardingComplete(ctx, userID); err != nil {
		h.log.Error("connect: check onboarding status", slog.String("error", err.Error()))
	} else if completed {
		redirectTo = "/settings?connected=1"
	}
	http.Redirect(w, r, redirectTo, http.StatusFound)
}
