-- Catalog and settings queries (ADR-001 §1.5, §4.4).

-- name: GetActiveCatalogPostByMedia :one
-- Cheap ingest filter: confirm media is registered & active before enqueuing.
SELECT * FROM catalog_post
WHERE ig_media_id = @ig_media_id
  AND account_id  = @account_id
  AND active      = true;

-- name: GetCatalogPostByID :one
SELECT * FROM catalog_post WHERE id = @id;

-- name: ListCatalogPostsByAccount :many
SELECT * FROM catalog_post
WHERE account_id = @account_id
ORDER BY created_at DESC;

-- Settings: per-account keyword & hold config (§4.4).

-- name: GetCommentOrderSettings :one
SELECT * FROM comment_order_settings WHERE account_id = @account_id;

-- name: UpsertCommentOrderSettings :one
-- Default keywords come from KIT_KEYWORDS.seller (packages/types/src/constants.ts).
-- Keep both in sync when updating defaults.
INSERT INTO comment_order_settings (account_id, keywords, hold_seconds, reply_template)
VALUES (@account_id, @keywords, @hold_seconds, @reply_template)
ON CONFLICT (account_id) DO UPDATE
    SET keywords       = EXCLUDED.keywords,
        hold_seconds   = EXCLUDED.hold_seconds,
        reply_template = EXCLUDED.reply_template
RETURNING *;
