// Package tasks contains asynq task handler implementations for the Zosmed worker.
// Task TYPE constants and payload structs live in libs/platform/tasks (shared with apps/api).
package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"github.com/zosmed/zosmed/libs/igapi"
	seller "github.com/zosmed/zosmed/libs/kits/seller"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/apps/worker/internal/runner"
)

// CommentIngestHandler handles the "comment:ingest" task (ptasks.TaskCommentIngest).
// It is the entry point for the comment-to-order flow inside the worker.
//
// Processing steps (ADR-001 §3.3):
//  1. Unmarshal CommentIngestPayload.
//  2. Run DetectKeepCode — skip if no match (not a keep/order comment).
//  3. Load catalog post — confirm media is registered & active for this account.
//  4. Load per-account comment-order settings (hold_seconds, reply_template).
//  5. Build workflow.Event with Raw context, call Engine.Run with igapi.Client
//     as the Sender (satisfies workflow.Sender structurally).
//
// Guardrail B: operates on catalog_post.ig_media_id (post/Reel). No IG Live ref.
// Guardrail F: outbound only via Engine → seller action → rc.Gate → rc.Sender.
type CommentIngestHandler struct {
	r      *runner.Runner
	logger *slog.Logger
}

// NewCommentIngestHandler constructs a handler bound to the given runner.
func NewCommentIngestHandler(r *runner.Runner, logger *slog.Logger) *CommentIngestHandler {
	return &CommentIngestHandler{r: r, logger: logger}
}

// ProcessTask implements the asynq handler signature.
func (h *CommentIngestHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ptasks.CommentIngestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("comment_ingest: unmarshal: %w", err)
	}

	log := h.logger.With(
		slog.String("comment_id", p.CommentID),
		slog.String("account_id", p.AccountID),
		slog.String("media_id", p.MediaID),
	)

	// Step 2: keep-code detection. Skip non-matching comments early.
	code, ok := seller.DetectKeepCode(p.Text)
	if !ok {
		log.Debug("comment_ingest: no keep code — skip")
		return nil
	}

	// Parse AccountID from string to pgtype.UUID.
	accountID, err := seller.ParseUUID(p.AccountID)
	if err != nil {
		return fmt.Errorf("comment_ingest: parse account_id %q: %w", p.AccountID, err)
	}

	// Step 3: confirm the media is a registered, active catalog post.
	catalogPost, err := h.r.DB.GetActiveCatalogPostByMedia(ctx, dbgen.GetActiveCatalogPostByMediaParams{
		IgMediaID: p.MediaID,
		AccountID: accountID,
	})
	if err != nil {
		if isNoRows(err) {
			log.Debug("comment_ingest: media not in active catalog — skip", slog.String("media_id", p.MediaID))
			return nil
		}
		return fmt.Errorf("comment_ingest: get catalog post: %w", err)
	}

	// Step 4: load per-account settings (non-fatal; defaults apply).
	holdSeconds := seller.DefaultHoldSeconds
	settings, settingsErr := h.r.DB.GetCommentOrderSettings(ctx, accountID)
	if settingsErr == nil {
		holdSeconds = settings.HoldSeconds
	}

	// The IG user ID of the business account (sender) comes from env for MVP.
	// In production this would be fetched from the accounts table per accountID.
	// Guardrail D: {nama} ONLY from webhook payload (p.FromUsername) — not scraped.
	igAccountUserID := os.Getenv("IG_ACCOUNT_USER_ID")

	// Step 5: build a source-agnostic workflow.Event and run the engine.
	// Raw carries seller-kit-specific context; engine is neutral to these keys.
	event := workflow.Event{
		Source:       workflow.SourceComment,
		AccountID:    p.AccountID,
		ObjectID:     p.CommentID,
		MediaID:      p.MediaID,
		FromID:       p.FromID,
		FromUsername: p.FromUsername,
		Text:         p.Text,
		Raw: map[string]any{
			seller.RawKeyCatalogPostID: seller.UUIDToString(catalogPost.ID),
			seller.RawKeyKode:          code,
			seller.RawKeyHoldSeconds:   holdSeconds,
			seller.RawKeyIgUserID:      igAccountUserID,
			seller.RawKeyCommentAt:     time.Now(), // approximate; used for 7-day window check
		},
	}

	// Create igapi.Client for this engine run.
	// igapi.Client satisfies workflow.Sender structurally (same method signatures).
	sender := igapi.New(h.r.IGToken)

	result, err := h.r.Engine.Run(ctx, event, sender, h.r.Gate)
	if err != nil {
		return fmt.Errorf("comment_ingest: engine run: %w", err)
	}

	log.Info("comment_ingest: engine run complete",
		slog.String("workflow_id", result.WorkflowID),
		slog.Bool("triggered", result.Triggered),
		slog.Bool("filter_passed", result.FilterPassed),
		slog.Int("steps", len(result.Steps)),
	)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// isNoRows returns true for pgx "no rows in result set" errors.
// Kept local to this package — acceptable duplication of a 2-line helper
// (§12a-4: not worth a shared package for this tiny function).
func isNoRows(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "no rows in result set")
}

