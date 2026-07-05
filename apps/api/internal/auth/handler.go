// Package auth implements Zosmed's own login identity — email+password,
// server-side sessions — layered on top of (and separate from) the Instagram
// `account` OAuth flow in apps/api/internal/connect (ADR-003).
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// minPasswordLen is the MVP password strength floor (no other complexity
// rules — hardening follow-up per ADR-003 §11).
const minPasswordLen = 8

// Handler implements the auth + onboarding HTTP endpoints (ADR-003 §3/§4).
type Handler struct {
	store        *Store
	secureCookie bool // true when APP_ENV=prod (cookie Secure flag)
	log          *slog.Logger
}

// New wires a Handler. secureCookie should be true only when serving over
// HTTPS in production (ADR-003 §0 decision 1).
func New(store *Store, secureCookie bool, log *slog.Logger) *Handler {
	return &Handler{store: store, secureCookie: secureCookie, log: log}
}

// ── POST /api/v1/auth/register ───────────────────────────────────────────────

// Register creates a new app_user and immediately logs them in (public route).
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Body permintaan tidak valid")
		return
	}

	email := normalizeEmail(body.Email)
	if !validEmail(email) {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Email tidak valid")
		return
	}
	if len(body.Password) < minPasswordLen {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Password minimal 8 karakter")
		return
	}
	// bcrypt rejects >72 bytes (ErrPasswordTooLong) — reject cleanly here so it
	// surfaces as 400, not a 500 from the hash step (review MINOR-2).
	if len(body.Password) > MaxPasswordLen {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Password maksimal 72 karakter")
		return
	}
	if body.Segment != nil && !ValidSegments[*body.Segment] {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Segmen tidak dikenal")
		return
	}

	hash, err := HashPassword(body.Password)
	if err != nil {
		h.log.Error("auth: hash password", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal mendaftarkan akun")
		return
	}

	user, err := h.store.CreateUser(ctx, email, hash)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpx.Err(w, http.StatusConflict, "email_taken", "Email sudah terdaftar")
			return
		}
		h.log.Error("auth: create user", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal mendaftarkan akun")
		return
	}

	if body.Segment != nil {
		user, err = h.store.SetSegment(ctx, user.ID, *body.Segment)
		if err != nil {
			h.log.Error("auth: set segment on register", slog.String("error", err.Error()))
			httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal menyimpan segmen")
			return
		}
	}

	if err := h.startSession(ctx, w, user.ID); err != nil {
		h.log.Error("auth: start session", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal membuat sesi login")
		return
	}

	httpx.JSON(w, http.StatusCreated, MeResponse{User: mapUserDTO(user)})
}

// ── POST /api/v1/auth/login ───────────────────────────────────────────────────

// Login verifies credentials and starts a new session (public route).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Body permintaan tidak valid")
		return
	}

	email := normalizeEmail(body.Email)

	user, err := h.store.UserByEmail(ctx, email)
	if err != nil {
		// Same response for "no such user" and "wrong password" — never leak
		// which one failed (ADR-003 §14: don't log passwords, don't leak state).
		if errors.Is(err, ErrNotFound) {
			// Burn one bcrypt comparison so an unknown email takes the same time
			// as a wrong password — closes the enumeration timing side-channel.
			VerifyDummyPassword(body.Password)
			h.log.Info("auth: login failed", slog.String("email", email), slog.String("reason", "invalid_credentials"))
			httpx.Err(w, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")
			return
		}
		h.log.Error("auth: login lookup", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal login")
		return
	}

	if !VerifyPassword(user.PasswordHash, body.Password) {
		h.log.Info("auth: login failed", slog.String("email", email), slog.String("reason", "invalid_credentials"))
		httpx.Err(w, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")
		return
	}

	if err := h.startSession(ctx, w, user.ID); err != nil {
		h.log.Error("auth: start session", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal membuat sesi login")
		return
	}

	httpx.JSON(w, http.StatusOK, MeResponse{
		User:    mapUserDTO(user),
		Account: h.loadAccountDTO(ctx, user.ID),
	})
}

// ── POST /api/v1/auth/logout ─────────────────────────────────────────────────

// Logout revokes the current session (if any) and clears the cookie. Public
// route: calling it without a session is a harmless no-op (ADR-003 §4.1).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := readSessionCookie(r); ok {
		if err := h.store.DeleteSession(r.Context(), hashToken(token)); err != nil {
			h.log.Error("auth: revoke session", slog.String("error", err.Error()))
			// Non-fatal: still clear the cookie client-side below.
		}
	}
	clearSessionCookie(w, h.secureCookie)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ── GET /api/v1/auth/me ───────────────────────────────────────────────────────

// Me returns the current session's user + linked account state. Requires
// RequireUser to have already populated the context.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userDTO, ok := UserFromContext(ctx)
	if !ok {
		// Should be unreachable: Me is always mounted behind RequireUser.
		httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Silakan login terlebih dahulu")
		return
	}

	userID, err := parseUserID(userDTO.ID)
	if err != nil {
		h.log.Error("auth: me parse user id", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal memuat akun")
		return
	}

	httpx.JSON(w, http.StatusOK, MeResponse{User: userDTO, Account: h.loadAccountDTO(ctx, userID)})
}

// ── helpers ───────────────────────────────────────────────────────────────────

// startSession mints a fresh session token, persists its hash, and sets the
// httpOnly cookie. Single place Register and Login both call (DRY §12a-1).
func (h *Handler) startSession(ctx context.Context, w http.ResponseWriter, userID pgtype.UUID) error {
	token, err := newSessionToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(sessionTTL)
	if err := h.store.CreateSession(ctx, hashToken(token), userID, expiresAt); err != nil {
		return err
	}
	setSessionCookie(w, token, h.secureCookie)
	return nil
}

// loadAccountDTO looks up the account linked to userID and maps it to the
// safe DTO. Returns nil (not an error) when no account is linked yet — that
// is the normal pre-connect state, not a failure (ADR-003 §4.3).
func (h *Handler) loadAccountDTO(ctx context.Context, userID pgtype.UUID) *AccountDTO {
	acc, err := h.store.AccountByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			h.log.Error("auth: load linked account", slog.String("error", err.Error()))
		}
		return nil
	}
	dto := mapAccountDTO(acc)
	return &dto
}

// parseUserID parses the string form of a user id (as stored in UserDTO,
// ADR-003 §4.3) back into a pgtype.UUID for store lookups.
func parseUserID(id string) (pgtype.UUID, error) {
	return uuidx.Parse(id)
}

// normalizeEmail lower-cases and trims an email for storage/lookup consistency
// (ADR-003 §2.2: "email UNIQUE ... disimpan lower-case").
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail is a minimal sanity check — full RFC 5322 validation is
// unnecessary for MVP; we just need to reject empty/obviously-malformed input.
func validEmail(email string) bool {
	if email == "" {
		return false
	}
	at := strings.IndexByte(email, '@')
	return at > 0 && at < len(email)-1 && !strings.ContainsAny(email, " \t\n")
}
