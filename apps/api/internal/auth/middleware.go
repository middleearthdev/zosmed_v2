package auth

import (
	"context"
	"net/http"

	"github.com/zosmed/zosmed/apps/api/internal/httpx"
)

type userContextKey struct{}

// WithUser stores the authenticated user's safe DTO in ctx. UserDTO is reused
// as the context-carried type (not a separate domain struct) — it is already
// the "safe" shape (no password_hash), and introducing a second near-identical
// struct just for context would be duplication without a real second concept
// (anti-over-abstraction §12a-4). Exported so sibling packages (connect's
// tests, ADR-003 §6) can build an authenticated context without a real cookie.
func WithUser(ctx context.Context, u UserDTO) context.Context {
	return context.WithValue(ctx, userContextKey{}, u)
}

// UserFromContext retrieves the user injected by RequireUser. Callers outside
// this package (e.g. connect.Handler.Start, ADR-003 §6) use this to identify
// the logged-in user for the OAuth connect flow.
func UserFromContext(ctx context.Context) (UserDTO, bool) {
	u, ok := ctx.Value(userContextKey{}).(UserDTO)
	return u, ok
}

// RequireUser builds session-auth middleware backed by store. It replaces the
// former no-op auth middleware (ADR-003 AC-6): requests without a valid "zsid"
// cookie/session get 401 {error:{code:"unauthorized"}} instead of passing through.
//
// A concrete *Store is used directly rather than an interface — there is only
// one implementation, so no AuthService abstraction is introduced (§12a-4).
func RequireUser(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := readSessionCookie(r)
			if !ok {
				httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Silakan login terlebih dahulu")
				return
			}

			u, err := store.SessionUser(r.Context(), hashToken(token))
			if err != nil {
				httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid atau sudah kedaluwarsa")
				return
			}

			ctx := WithUser(r.Context(), mapUserDTO(u))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
