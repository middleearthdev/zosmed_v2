-- Dedupe ledger for ingest (ADR-001 §3.2 step 3 / §8 — "Dedupe: dua lapis").
-- InsertProcessedComment returns rows affected; caller checks == 0 → already processed.

-- name: InsertProcessedComment :execrows
INSERT INTO processed_comment (
    ig_comment_id,
    account_id,
    ig_media_id,
    comment_text,
    contact_ig_user_id,
    contact_handle
) VALUES (
    @ig_comment_id,
    @account_id,
    @ig_media_id,
    @comment_text,
    @contact_ig_user_id,
    @contact_handle
)
ON CONFLICT (ig_comment_id) DO NOTHING;

-- name: ExistsProcessedComment :one
-- Read-check for enqueue-first ordering (ADR-007 §2.2 step 2): lets the webhook
-- handler skip re-delivered events BEFORE attempting to enqueue, and catches
-- Meta retries that arrive after the asynq TaskID Retention window has expired.
SELECT EXISTS (
    SELECT 1 FROM processed_comment WHERE ig_comment_id = @ig_comment_id
);
