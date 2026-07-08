-- Dedupe ledger for messaging ingest (ADR-006 §5.4, mirror comments.sql).

-- name: InsertProcessedMessage :execrows
-- 0 rows → sudah pernah diproses (dedupe). Pola sama InsertProcessedComment.
INSERT INTO processed_message (ig_message_id, account_id, subtype, contact_ig_user_id)
VALUES (@ig_message_id, @account_id, @subtype, @contact_ig_user_id)
ON CONFLICT (ig_message_id) DO NOTHING;
