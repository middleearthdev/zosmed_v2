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
