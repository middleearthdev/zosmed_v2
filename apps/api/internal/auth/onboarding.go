package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/zosmed/zosmed/apps/api/internal/httpx"
)

// accountConnectedStatus is the account.status value meaning "fully usable"
// (mirrors db/migrations/00001_accounts.sql's status CHECK). Onboarding
// cannot complete until the linked account reaches this state (ADR-003 §4.2).
const accountConnectedStatus = "connected"

// ── PUT /api/v1/onboarding/segment ───────────────────────────────────────────

// PutSegment persists the user's chosen business segment
// (seller|creator|booking). Requires RequireUser.
func (h *Handler) PutSegment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userDTO, ok := UserFromContext(ctx)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Silakan login terlebih dahulu")
		return
	}

	var body SegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Body permintaan tidak valid")
		return
	}
	if !ValidSegments[body.Segment] {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Segmen tidak dikenal (gunakan seller/creator/booking)")
		return
	}

	userID, err := parseUserID(userDTO.ID)
	if err != nil {
		h.log.Error("auth: put segment parse user id", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal menyimpan segmen")
		return
	}

	updated, err := h.store.SetSegment(ctx, userID, body.Segment)
	if err != nil {
		h.log.Error("auth: set segment", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal menyimpan segmen")
		return
	}

	httpx.JSON(w, http.StatusOK, MeResponse{User: mapUserDTO(updated)})
}

// ── POST /api/v1/onboarding/complete ─────────────────────────────────────────

// CompleteOnboarding stamps onboarding_completed_at once both preconditions
// hold: segment chosen AND Instagram account connected (ADR-003 §4.2). Requires
// RequireUser.
func (h *Handler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userDTO, ok := UserFromContext(ctx)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Silakan login terlebih dahulu")
		return
	}

	userID, err := parseUserID(userDTO.ID)
	if err != nil {
		h.log.Error("auth: complete onboarding parse user id", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal menyelesaikan onboarding")
		return
	}

	if userDTO.Segment == nil {
		httpx.ErrWithReason(w, http.StatusConflict, "onboarding_incomplete",
			"Pilih segmen bisnis kamu terlebih dahulu", "segment_missing")
		return
	}

	account := h.loadAccountDTO(ctx, userID)
	if account == nil || account.Status != accountConnectedStatus {
		httpx.ErrWithReason(w, http.StatusConflict, "onboarding_incomplete",
			"Hubungkan akun Instagram kamu terlebih dahulu", "account_not_connected")
		return
	}

	updated, err := h.store.CompleteOnboarding(ctx, userID)
	if err != nil {
		h.log.Error("auth: complete onboarding", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "Gagal menyelesaikan onboarding")
		return
	}

	httpx.JSON(w, http.StatusOK, MeResponse{User: mapUserDTO(updated)})
}
