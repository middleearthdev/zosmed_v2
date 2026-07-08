package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/worker/internal/runner"
	"github.com/zosmed/zosmed/apps/worker/internal/wfload"
	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
	"github.com/zosmed/zosmed/libs/workflow"
)

// Event.Raw keys this handler writes for every messaging event — MUST stay
// identical to the string literals duplicated in libs/workflow/nodes
// (rawKeyIgUserID/rawKeyEventSubtype/rawKeyLastInteractionAt/rawKeyAdRef,
// action_wa_link.go/trigger_dm.go — ADR-006 §9 wire-key convention, §12a-4).
const (
	rawKeyIgUserID          = "ig_user_id"
	rawKeyEventSubtype      = "event_subtype"
	rawKeyLastInteractionAt = "last_interaction_at"
	rawKeyAdRef             = "ad_ref"
)

// dmIngestLastSource is the value written to conversation.last_source for
// every event this handler processes — all messaging-surface events are
// "dm" (ADR-006 koreksi B0 point 4; §5.1 comment).
const dmIngestLastSource = "dm"

// DMIngestHandler handles the "dm:ingest" task (ptasks.TaskDMIngest) — the
// entry point for the messaging/story ingest pipeline (ADR-006 §4.1).
//
// Mirrors CommentIngestHandler's structure but WITHOUT the catalog_post/
// seller-kit enrichment (DM/story is not seller-specific, §12a-3 SoC) and
// WITHOUT the legacy built-in-workflow fallback (comment_ingest.go's R3 —
// the DM path has no such legacy to fall back to).
//
// Processing steps (ADR-006 §4.1):
//  1. Unmarshal DMIngestPayload.
//  2. Load the connected account; skip if not 'connected'.
//  3. Upsert conversation.last_interaction_at (window store, §4.2) — EVERY
//     messaging event (DM/story-reply/story-mention/ad-referral) opens or
//     refreshes the 24h window (R3).
//  4. Build workflow.Event{Source: dm, Raw: {...}}.
//  5. Load→compile→run every `live` workflow for the account; the first
//     Triggered wins (RunStore.Insert), mirroring comment_ingest.go.
type DMIngestHandler struct {
	r      *runner.Runner
	logger *slog.Logger
}

// NewDMIngestHandler constructs a handler bound to the given runner.
func NewDMIngestHandler(r *runner.Runner, logger *slog.Logger) *DMIngestHandler {
	return &DMIngestHandler{r: r, logger: logger}
}

// ProcessTask implements the asynq handler signature.
func (h *DMIngestHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ptasks.DMIngestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("dm_ingest: unmarshal: %w", err)
	}

	log := h.logger.With(
		slog.String("message_id", p.MessageID),
		slog.String("account_id", p.AccountID),
		slog.String("subtype", p.Subtype),
	)

	accountID, err := uuidx.Parse(p.AccountID)
	if err != nil {
		return fmt.Errorf("dm_ingest: parse account_id %q: %w", p.AccountID, err)
	}

	// Load the connected account (mirrors comment_ingest.go — token + ig_user_id
	// always come from Postgres, never the task payload, ADR-002 §6.2).
	account, err := h.r.DB.GetAccountByID(ctx, accountID)
	if err != nil {
		if isNoRows(err) {
			log.Warn("dm_ingest: unknown account — skip")
			return nil
		}
		return fmt.Errorf("dm_ingest: get account: %w", err)
	}
	if account.Status != "connected" {
		log.Warn("dm_ingest: account not connected — skip",
			slog.String("status", account.Status),
		)
		return nil
	}

	// Resolve the event time (ADR-006 §4.1 step 4): webhook timestamp, falling
	// back to now for legacy/empty payloads.
	eventAt := time.Now().UTC()
	if p.EventAt != "" {
		if parsed, perr := time.Parse(time.RFC3339, p.EventAt); perr == nil {
			eventAt = parsed
		}
	}

	// WINDOW (ADR-006 §4.1 step 4 / R3): every messaging event — DM,
	// story-reply, story-mention, AND ad-referral — opens/refreshes the 24h
	// window. p.Source is always "dm" for this whole surface, so this upsert
	// runs unconditionally (no branch needed for story-mention, unlike the
	// rejected draft premise, R3).
	conv, err := h.r.DB.UpsertConversationInteraction(ctx, dbgen.UpsertConversationInteractionParams{
		AccountID:         accountID,
		ContactIgUserID:   p.FromID,
		LastInteractionAt: pgtype.Timestamptz{Time: eventAt, Valid: true},
		LastSource:        dmIngestLastSource,
	})
	if err != nil {
		return fmt.Errorf("dm_ingest: upsert conversation: %w", err)
	}
	lastInteraction := conv.LastInteractionAt.Time

	// Build a source-agnostic workflow.Event. Raw carries the messaging
	// subtype (trigger discriminator), the always-populated window timestamp
	// (send-dm's guard/gate input), and the sender ig_user_id — all consumed
	// by libs/workflow/nodes (dm-received/story-reply/story-mention/
	// click-to-dm-ad/conversation-state/send-dm, ADR-006 §2).
	raw := map[string]any{
		rawKeyIgUserID:          account.IgUserID,
		rawKeyEventSubtype:      p.Subtype,
		rawKeyLastInteractionAt: lastInteraction,
	}
	if p.AdRef != "" {
		raw[rawKeyAdRef] = p.AdRef
	}
	event := workflow.Event{
		Source:       workflow.SourceDM,
		AccountID:    p.AccountID,
		ObjectID:     p.MessageID,
		MediaID:      p.MediaID,
		FromID:       p.FromID,
		FromUsername: p.FromUsername,
		Text:         p.Text,
		Raw:          raw,
	}

	sender := igapi.New(account.AccessToken)
	triggerSummary := fmt.Sprintf("DM by @%s", p.FromUsername)
	runStart := time.Now()

	loaded, err := h.r.Loader.LoadLive(ctx, accountID)
	if err != nil {
		return fmt.Errorf("dm_ingest: load live workflows: %w", err)
	}
	if len(loaded) == 0 {
		// No legacy built-in fallback for DM/story (unlike comment_ingest.go's
		// R3) — the messaging path only ever runs persisted `live` workflows.
		log.Debug("dm_ingest: no live workflow — skip")
		return nil
	}

	for _, lw := range loaded {
		reg, def, err := h.r.Compiler.Compile(lw.PWF)
		if err != nil {
			log.Error("dm_ingest: compile workflow — skipped",
				slog.String("workflow_name", lw.Name),
				slog.String("error", err.Error()),
			)
			continue
		}

		eng := workflow.NewEngine(reg, []workflow.WorkflowDef{def})
		result, err := eng.Run(ctx, event, sender, h.r.Gate)
		if err != nil {
			return fmt.Errorf("dm_ingest: engine run (workflow %q): %w", lw.Name, err)
		}
		if !result.Triggered {
			continue
		}

		logEngineResult(log, lw.Name, result)

		workflowID, parseErr := uuidx.Parse(lw.PWF.ID)
		if parseErr != nil {
			log.Error("dm_ingest: parse workflow id", slog.String("error", parseErr.Error()))
		} else if err := h.r.RunStore.Insert(ctx, result, wfload.RunMeta{
			AccountID:      accountID,
			WorkflowID:     &workflowID,
			WorkflowName:   lw.Name,
			TriggerSource:  workflow.SourceDM,
			TriggerSummary: triggerSummary,
			ObjectID:       p.MessageID,
			DurationMs:     int32(time.Since(runStart).Milliseconds()),
		}); err != nil {
			log.Error("dm_ingest: insert run log", slog.String("error", err.Error()))
		}
		return nil
	}

	log.Debug("dm_ingest: no live workflow triggered")
	return nil
}
