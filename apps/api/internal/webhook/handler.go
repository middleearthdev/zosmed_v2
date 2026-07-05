package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/api/internal/enqueue"
	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/tasks"
)

// Handler handles Instagram webhook requests for the Zosmed API server
// (webhooks are subscribed via the Instagram product, CLAUDE.md §4.0/§6.4).
//
// Responsibilities (ADR-001 §3.2; account resolution per ADR-002 §6.1):
//   - GET /webhooks/meta  → verify webhook challenge handshake
//   - POST /webhooks/meta → verify signature → resolve account (IGSID) → dedupe → filter → enqueue
//
// This handler DOES NOT call the Instagram API, write reservations, or
// perform any heavy processing. All of that happens in apps/worker (SoC §12a-3).
type Handler struct {
	queries     *dbgen.Queries
	enq         *enqueue.Client
	appSecret   string
	verifyToken string
	log         *slog.Logger
}

// New returns a Handler wired with its dependencies. Account resolution is
// no longer a startup-time static value (ADR-002 §6.1) — each webhook entry
// carries its own IGSID (entry.id), looked up per request via GetAccountByIgUserID.
func New(
	queries *dbgen.Queries,
	enq *enqueue.Client,
	appSecret, verifyToken string,
	log *slog.Logger,
) *Handler {
	return &Handler{
		queries:     queries,
		enq:         enq,
		appSecret:   appSecret,
		verifyToken: verifyToken,
		log:         log,
	}
}

// Challenge handles GET /webhooks/meta for Meta webhook subscription verification.
// On success it writes hub.challenge as plain text with 200.
// On mismatch it responds with 403.
func (h *Handler) Challenge(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode := q.Get("hub.mode")
	token := q.Get("hub.verify_token")
	challenge := q.Get("hub.challenge")

	c, ok := VerifyChallenge(h.verifyToken, mode, token, challenge)
	if !ok {
		h.log.Warn("webhook: challenge verify failed",
			slog.String("mode", mode),
		)
		httpx.Err(w, http.StatusForbidden, "invalid_verify_token", "verify token mismatch")
		return
	}
	// Meta expects the raw challenge string (no JSON wrapping).
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(c))
}

// Receive handles POST /webhooks/meta for incoming webhook events from Meta.
//
// Processing pipeline (ADR-001 §3.2):
//  1. Read raw body BEFORE unmarshal (HMAC needs raw bytes).
//  2. Verify X-Hub-Signature-256 — 403 on failure, no processing.
//  3. Unmarshal payload; extract comment events.
//  4. Per comment: dedupe via processed_comment (ON CONFLICT DO NOTHING; 0 rows → skip).
//  5. Filter: skip enqueue if media_id not in active catalog.
//  6. EnqueueCommentIngest for the worker.
//  7. Respond 200 ASAP — processing is asynchronous.
func (h *Handler) Receive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 1: read raw body before any parsing.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("webhook: read body", slog.String("error", err.Error()))
		httpx.Err(w, http.StatusBadRequest, "read_body", "cannot read request body")
		return
	}

	// Step 2: verify X-Hub-Signature-256 with constant-time HMAC comparison.
	sig := r.Header.Get("X-Hub-Signature-256")
	if !VerifySignature(body, sig, h.appSecret) {
		h.log.Warn("webhook: signature verification failed")
		httpx.Err(w, http.StatusForbidden, "invalid_signature", "signature mismatch")
		return
	}

	// Step 3: unmarshal payload.
	var payload MetaPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.log.Error("webhook: unmarshal payload", slog.String("error", err.Error()))
		// Respond 200 anyway — if Meta sends a malformed body, retrying won't fix it.
		httpx.JSON(w, http.StatusOK, map[string]bool{"received": true})
		return
	}

	// Steps 4–6: process each comment event.
	comments := ExtractComments(payload)
	for _, c := range comments {
		if err := h.processComment(ctx, c); err != nil {
			// Log but never return non-200: Meta retries on failures, causing duplicate enqueue.
			h.log.Error("webhook: process comment",
				slog.String("error", err.Error()),
				slog.String("comment_id", c.Value.ID),
			)
		}
	}

	// Step 7: acknowledge immediately.
	httpx.JSON(w, http.StatusOK, map[string]bool{"received": true})
}

// processComment resolves the account, then performs dedupe, catalog filter,
// and enqueue for one comment event.
// Returns nil on skip (unknown account / duplicate / not in catalog) or
// successful enqueue. Returns a non-nil error only on DB or queue failures
// worth logging.
func (h *Handler) processComment(ctx context.Context, ic IngestComment) error {
	v := ic.Value
	if v.ID == "" || v.Media.ID == "" {
		return nil // skip incomplete events
	}

	// Step 4 (ADR-002 §6.1): resolve the connected account from entry.id
	// (IGSID). Comments belonging to an account Zosmed doesn't know about are
	// skipped safely — never a 500, since Meta would retry forever otherwise.
	account, err := h.queries.GetAccountByIgUserID(ctx, ic.EntryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Genuinely unknown account → skip safely (never a 500).
			h.log.Debug("webhook: unknown account for entry — skip",
				slog.String("entry_id", ic.EntryID),
				slog.String("comment_id", v.ID),
			)
			return nil
		}
		// A real DB failure (down/transient) must not be silently swallowed as
		// "unknown account" — surface it so Receive logs it at Error level.
		return fmt.Errorf("webhook: resolve account: %w", err)
	}
	accountID := account.ID

	// Step 5: dedupe ledger. ON CONFLICT DO NOTHING → 0 rows = already processed.
	rows, err := h.queries.InsertProcessedComment(ctx, dbgen.InsertProcessedCommentParams{
		IgCommentID:     v.ID,
		AccountID:       accountID,
		IgMediaID:       v.Media.ID,
		CommentText:     v.Text,
		ContactIgUserID: v.From.ID,
		ContactHandle:   v.From.Username,
	})
	if err != nil {
		return fmt.Errorf("webhook: insert processed comment: %w", err)
	}
	if rows == 0 {
		h.log.Debug("webhook: duplicate comment — skip",
			slog.String("comment_id", v.ID),
		)
		return nil
	}

	// Step 6: cheap filter — only enqueue if the media is in an active catalog post.
	_, err = h.queries.GetActiveCatalogPostByMedia(ctx, dbgen.GetActiveCatalogPostByMediaParams{
		IgMediaID: v.Media.ID,
		AccountID: accountID,
	})
	if err != nil {
		// pgx "no rows" or any other error means media is not registered / not active.
		h.log.Debug("webhook: media not in active catalog — skip enqueue",
			slog.String("media_id", v.Media.ID),
		)
		return nil
	}

	// Capture the comment time now (M4): the webhook entry timestamp, or the
	// receipt time if the entry omitted it — both are far closer to the real
	// comment time than the worker's dequeue time would be.
	commentAt := time.Now().UTC()
	if ic.EntryTime > 0 {
		commentAt = time.Unix(ic.EntryTime, 0).UTC()
	}

	// Step 7: enqueue for worker (heavy lifting: keep-code detect, reserve, reply).
	// Only the account UUID goes into the payload — the access token stays in
	// Postgres; the worker looks it up itself (ADR-002 §6.2, never in Redis).
	if err := h.enq.EnqueueCommentIngest(ctx, tasks.CommentIngestPayload{
		AccountID:    uuidToString(accountID),
		CommentID:    v.ID,
		MediaID:      v.Media.ID,
		FromID:       v.From.ID,
		FromUsername: v.From.Username,
		Text:         v.Text,
		CommentAt:    commentAt.Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("webhook: enqueue comment ingest: %w", err)
	}

	h.log.Info("webhook: comment enqueued",
		slog.String("comment_id", v.ID),
		slog.String("media_id", v.Media.ID),
	)
	return nil
}

// uuidToString formats a pgtype.UUID as a lowercase hyphenated UUID string.
// Package-local helper — avoids pulling seller kit into the webhook transport layer.
func uuidToString(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
