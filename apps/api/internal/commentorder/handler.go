package commentorder

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	seller "github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

const (
	// defaultHoldSeconds mirrors seller.DefaultHoldSeconds as the REST-layer fallback.
	// Not re-defined here — imported from the single source (§12a-1 DRY).
	defaultLimitComments = int32(50) // max comments returned per request
	defaultLimitReserv   = int32(50) // max reservations returned per request
)

// Handler handles the Comment-to-Order REST endpoints (ADR-001 §4.2–§4.4).
// It reads normalised data from dbgen and maps to typed DTOs.
// It does NOT run the workflow engine, send IG messages, or import heavy kit logic —
// those concerns live in apps/worker.
type Handler struct {
	queries *dbgen.Queries
}

// NewHandler returns a Handler backed by the given Queries.
func NewHandler(q *dbgen.Queries) *Handler {
	return &Handler{queries: q}
}

// ── GET /api/v1/comment-order?accountId=&postId= ──────────────────────────────

// GetCommentOrder returns the aggregated Comment-to-Order screen data.
// Query params: accountId (Postgres UUID), postId (catalog_post.id UUID).
func (h *Handler) GetCommentOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountID, ok := parseUUIDParam(w, r.URL.Query().Get("accountId"), "accountId")
	if !ok {
		return
	}
	postID, ok := parseUUIDParam(w, r.URL.Query().Get("postId"), "postId")
	if !ok {
		return
	}

	// Resolve the catalog post to get the ig_media_id string (needed for
	// ListRecentCommentsByPost which is keyed by ig_media_id, not the UUID).
	post, err := h.queries.GetCatalogPostByID(ctx, postID)
	if err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusNotFound, "post_not_found", "catalog post not found")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load catalog post")
		return
	}

	// Fan-out: 5 queries in sequence for MVP (no goroutine fan-out to keep error
	// handling readable; latency is acceptable for a dashboard screen).

	commentRows, err := h.queries.ListRecentCommentsByPost(ctx, dbgen.ListRecentCommentsByPostParams{
		AccountID: accountID,
		IgMediaID: post.IgMediaID,
		Lim:       defaultLimitComments,
	})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load comments")
		return
	}

	stats, err := h.queries.GetCommentOrderStats(ctx, postID)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load stats")
		return
	}

	commentCount, err := h.queries.GetPostCommentCount(ctx, postID)
	if err != nil && !isNoRows(err) {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load comment count")
		return
	}

	reservRows, err := h.queries.ListReservationsByPostWithProduct(ctx, dbgen.ListReservationsByPostWithProductParams{
		CatalogPostID: postID,
		Off:           0,
		Lim:           defaultLimitReserv,
	})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load reservations")
		return
	}

	productRows, err := h.queries.ListProductsByPost(ctx, postID)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load products")
		return
	}

	// ── Map to DTOs (single mapping location — DRY §12a-1) ──────────────────

	comments := make([]IncomingCommentDTO, 0, len(commentRows))
	for _, row := range commentRows {
		comments = append(comments, mapIncomingComment(row))
	}

	reservations := make([]ReservationDTO, 0, len(reservRows))
	for _, row := range reservRows {
		reservations = append(reservations, mapReservationFromJoin(row))
	}

	products := make([]CatalogProductDTO, 0, len(productRows))
	for _, row := range productRows {
		products = append(products, CatalogProductDTO{
			Code:       row.Code,
			Name:       row.Name,
			StockLeft:  row.StockLeft,
			StockTotal: row.StockTotal,
		})
	}

	httpx.JSON(w, http.StatusOK, CommentOrderResponse{
		PostCommentsLabel: formatCommentCount(commentCount),
		Comments:          comments,
		Stats:             mapStats(stats),
		Reservations:      reservations,
		Products:          products,
	})
}

// ── POST /api/v1/reservations/{id}/close ──────────────────────────────────────

// CloseReservation transitions a reservation from waiting-pay → closed-wa.
// Only valid for reservations currently in waiting-pay; returns 409 otherwise.
func (h *Handler) CloseReservation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resID, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	updated, err := h.queries.UpdateReservationStatus(ctx, dbgen.UpdateReservationStatusParams{
		NewStatus:      dbgen.ReservationStatusClosedWa,
		ClosedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ID:             resID,
		ExpectedStatus: dbgen.ReservationStatusWaitingPay,
	})
	if err != nil {
		if isNoRows(err) {
			// Guard fired: reservation not in waiting-pay (may be reserved, closed, or expired).
			httpx.Err(w, http.StatusConflict, "invalid_transition",
				"reservation is not in waiting-pay state; cannot close")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to update reservation")
		return
	}

	// Fetch product info for the DTO price label.
	prod, err := h.queries.GetProductByPostAndCode(ctx, dbgen.GetProductByPostAndCodeParams{
		CatalogPostID: updated.CatalogPostID,
		Code:          updated.Code,
	})
	if err != nil {
		// Non-fatal: return a partial DTO without price.
		httpx.JSON(w, http.StatusOK, mapReservationNoProduct(updated))
		return
	}

	httpx.JSON(w, http.StatusOK, mapReservationFromModel(updated, prod))
}

// ── GET /api/v1/reservations/{id} ────────────────────────────────────────────

// GetReservation returns the detail of one reservation by its Postgres UUID.
func (h *Handler) GetReservation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resID, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	res, err := h.queries.GetReservation(ctx, resID)
	if err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusNotFound, "not_found", "reservation not found")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load reservation")
		return
	}

	prod, err := h.queries.GetProductByPostAndCode(ctx, dbgen.GetProductByPostAndCodeParams{
		CatalogPostID: res.CatalogPostID,
		Code:          res.Code,
	})
	if err != nil {
		httpx.JSON(w, http.StatusOK, mapReservationNoProduct(res))
		return
	}

	httpx.JSON(w, http.StatusOK, mapReservationFromModel(res, prod))
}

// ── GET /api/v1/comment-order/settings?accountId= ────────────────────────────

// GetSettings returns the per-account comment-order keyword settings.
// When no row exists, returns default seller keywords (§12a-1: single source
// = seller.KitKeywords).
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountID, ok := parseUUIDParam(w, r.URL.Query().Get("accountId"), "accountId")
	if !ok {
		return
	}

	s, err := h.queries.GetCommentOrderSettings(ctx, accountID)
	if err != nil {
		if isNoRows(err) {
			// No row yet — return defaults.
			httpx.JSON(w, http.StatusOK, defaultSettings())
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to load settings")
		return
	}

	httpx.JSON(w, http.StatusOK, SettingsDTO{
		Keywords:      s.Keywords,
		HoldSeconds:   s.HoldSeconds,
		ReplyTemplate: s.ReplyTemplate,
	})
}

// ── PUT /api/v1/comment-order/settings ───────────────────────────────────────

// PutSettings upserts the per-account comment-order keyword settings.
// Expects JSON body: SettingsDTO. accountId in query param identifies the account.
func (h *Handler) PutSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountID, ok := parseUUIDParam(w, r.URL.Query().Get("accountId"), "accountId")
	if !ok {
		return
	}

	var body SettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_body", "cannot parse request body")
		return
	}

	// Apply defaults for empty fields so the DB row is always valid.
	if len(body.Keywords) == 0 {
		body.Keywords = seller.KitKeywords
	}
	if body.HoldSeconds <= 0 {
		body.HoldSeconds = seller.DefaultHoldSeconds
	}

	updated, err := h.queries.UpsertCommentOrderSettings(ctx, dbgen.UpsertCommentOrderSettingsParams{
		AccountID:     accountID,
		Keywords:      body.Keywords,
		HoldSeconds:   body.HoldSeconds,
		ReplyTemplate: body.ReplyTemplate,
	})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "failed to save settings")
		return
	}

	httpx.JSON(w, http.StatusOK, SettingsDTO{
		Keywords:      updated.Keywords,
		HoldSeconds:   updated.HoldSeconds,
		ReplyTemplate: updated.ReplyTemplate,
	})
}

// ── Mapping helpers (single location — DRY §12a-1) ───────────────────────────

// mapIncomingComment maps one ListRecentCommentsByPostRow to IncomingCommentDTO.
func mapIncomingComment(row dbgen.ListRecentCommentsByPostRow) IncomingCommentDTO {
	reserved := row.ReservationID.Valid &&
		row.ReservationStatus != nil &&
		*row.ReservationStatus != dbgen.ReservationStatusExpiredReleased

	return IncomingCommentDTO{
		ID:          row.IgCommentID,
		User:        row.ContactHandle,
		Text:        row.CommentText,
		Ago:         agoLabel(row.ReceivedAt),
		MatchedCode: row.MatchedCode,
		Reserved:    reserved,
		Duplicate:   false, // deduped at ingest layer; all rows in processed_comment are unique
	}
}

// mapReservationFromJoin maps a ListReservationsByPostWithProductRow to ReservationDTO.
// Price and product name come from the JOIN — no extra round-trip needed.
func mapReservationFromJoin(row dbgen.ListReservationsByPostWithProductRow) ReservationDTO {
	return ReservationDTO{
		ID:             uuidToString(row.ID),
		Code:           row.Code,
		BuyerHandle:    row.ContactHandle,
		Product:        row.ProductName,
		PriceLabel:     formatPriceLabel(row.ProductPriceIdr),
		Status:         string(row.Status),
		CountdownLabel: countdownLabel(row.Status),
		ExpiresAt:      row.ExpiresAt.Time,
	}
}

// mapReservationFromModel maps a Reservation + Product to ReservationDTO.
// Used for single-reservation endpoints where the JOIN data isn't available.
func mapReservationFromModel(res dbgen.Reservation, prod dbgen.Product) ReservationDTO {
	return ReservationDTO{
		ID:             uuidToString(res.ID),
		Code:           res.Code,
		BuyerHandle:    res.ContactHandle,
		Product:        prod.Name,
		PriceLabel:     formatPriceLabel(prod.PriceIdr),
		Status:         string(res.Status),
		CountdownLabel: countdownLabel(res.Status),
		ExpiresAt:      res.ExpiresAt.Time,
	}
}

// mapReservationNoProduct maps a Reservation to ReservationDTO without product info.
// Used as a fallback when the product lookup fails (non-fatal path).
func mapReservationNoProduct(res dbgen.Reservation) ReservationDTO {
	return ReservationDTO{
		ID:             uuidToString(res.ID),
		Code:           res.Code,
		BuyerHandle:    res.ContactHandle,
		Product:        "—",
		PriceLabel:     "Rp —",
		Status:         string(res.Status),
		CountdownLabel: countdownLabel(res.Status),
		ExpiresAt:      res.ExpiresAt.Time,
	}
}

// mapStats converts the aggregate stats row to the four stat tiles.
// stat key values are the canonical identifiers expected by the FE (§4.2).
//
// N9: TotalDetected counts reservations created — i.e. keep codes detected that
// HAD stock. Keep comments that matched a code but were out of stock create no
// reservation and are not counted, so the label says "ter-reserve" (reserved),
// not "terdeteksi" (detected), to avoid overstating detection coverage.
func mapStats(s dbgen.GetCommentOrderStatsRow) []CommentOrderStatDTO {
	return []CommentOrderStatDTO{
		{Key: "code-detected", Label: "Kode ter-reserve", Value: fmt.Sprintf("%d", s.TotalDetected)},
		{Key: "reserved-now", Label: "Reserved sekarang", Value: fmt.Sprintf("%d", s.ReservedNow)},
		{Key: "closed-wa", Label: "Closed via WA", Value: fmt.Sprintf("%d", s.ClosedWa)},
		{Key: "expired", Label: "Expired/dilepas", Value: fmt.Sprintf("%d", s.ExpiredReleased)},
	}
}

// ── Format helpers (all defined once here, used only in this package) ─────────

// formatPriceLabel converts a price in IDR to an olshop-style display label.
//
//	189000  → "Rp 189rb"
//	2000000 → "Rp 2jt"
//	1500000 → "Rp 1.5jt"
//	500     → "Rp 500"
func formatPriceLabel(priceIDR int64) string {
	switch {
	case priceIDR <= 0:
		return "Rp —"
	case priceIDR < 1_000:
		return fmt.Sprintf("Rp %d", priceIDR)
	case priceIDR < 1_000_000:
		return fmt.Sprintf("Rp %drb", priceIDR/1_000)
	default:
		whole := priceIDR / 1_000_000
		frac := (priceIDR % 1_000_000) / 100_000 // one decimal digit
		if frac == 0 {
			return fmt.Sprintf("Rp %djt", whole)
		}
		return fmt.Sprintf("Rp %d.%djt", whole, frac)
	}
}

// countdownLabel returns the countdown display string for a reservation status.
// Active statuses (reserved, waiting-pay) return "—" so the FE uses ExpiresAt
// for the live countdown. Terminal statuses return a static label.
func countdownLabel(status dbgen.ReservationStatus) string {
	switch status {
	case dbgen.ReservationStatusClosedWa:
		return "✓ closed"
	case dbgen.ReservationStatusExpiredReleased:
		return "— released"
	default:
		// reserved or waiting-pay: FE calculates live countdown from expiresAt.
		return "—"
	}
}

// agoLabel returns a human-readable relative time in Bahasa Indonesia.
func agoLabel(t pgtype.Timestamptz) string {
	if !t.Valid {
		return "—"
	}
	d := time.Since(t.Time)
	switch {
	case d < time.Minute:
		return "baru saja"
	case d < time.Hour:
		return fmt.Sprintf("%d mnt lalu", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d jam lalu", int(d.Hours()))
	default:
		return fmt.Sprintf("%d hari lalu", int(d.Hours()/24))
	}
}

// formatCommentCount formats the total comment count for the post header label.
func formatCommentCount(n int32) string {
	return fmt.Sprintf("%d komentar", n)
}

// defaultSettings returns the system default keyword settings for a new account.
// Keywords come from seller.KitKeywords — single source (§12a-1 DRY).
func defaultSettings() SettingsDTO {
	return SettingsDTO{
		Keywords:      seller.KitKeywords,
		HoldSeconds:   seller.DefaultHoldSeconds,
		ReplyTemplate: "",
	}
}

// ── UUID helpers ──────────────────────────────────────────────────────────────

// parseUUIDParam parses a UUID string from a query param / path param.
// On parse failure it writes a 400 response and returns false so callers can
// do an early return.
func parseUUIDParam(w http.ResponseWriter, raw, paramName string) (pgtype.UUID, bool) {
	id, err := uuidx.Parse(raw)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_param",
			fmt.Sprintf("invalid UUID for param %q: %s", paramName, raw))
		return pgtype.UUID{}, false
	}
	return id, true
}

// uuidToString formats a pgtype.UUID as a lowercase hyphenated UUID string.
// Uses seller.UUIDToString (single source for UUID formatting — §12a-1).
func uuidToString(u pgtype.UUID) string {
	return uuidx.Format(u)
}

// ── DB error helpers ──────────────────────────────────────────────────────────

// isNoRows returns true when err represents "no rows in result set" from pgx.
// Matches pgx.ErrNoRows directly (preferred over string matching where possible).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || (err != nil && strings.HasSuffix(err.Error(), "no rows in result set"))
}
