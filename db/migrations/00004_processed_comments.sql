-- +goose Up
-- processed_comment: dedupe ledger for ingest (ADR-001 §3.2 step 3).
-- Primary function: InsertProcessedComment ON CONFLICT DO NOTHING; caller checks rows_affected.
-- Extended columns (ig_media_id, comment_text, contact_*) serve the ListRecentCommentsByPost
-- UI query without a separate comment-log table; all fields have safe defaults so the
-- minimal dedupe insert still works.
CREATE TABLE processed_comment (
    ig_comment_id        text        PRIMARY KEY,
    account_id           uuid        NOT NULL REFERENCES account(id),
    ig_media_id          text        NOT NULL DEFAULT '',  -- used for per-post UI filtering
    comment_text         text        NOT NULL DEFAULT '',
    contact_ig_user_id   text        NOT NULL DEFAULT '',
    contact_handle       text        NOT NULL DEFAULT '',
    received_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX processed_comment_account_media_idx
    ON processed_comment(account_id, ig_media_id, received_at DESC);

-- +goose Down
DROP TABLE processed_comment;
