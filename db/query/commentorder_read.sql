-- Aggregate read queries for the Comment-to-Order screen (ADR-001 §4.2).
-- All queries keyed by catalog_post_id or account_id; designed for a single
-- GET /api/v1/comment-order?accountId=&postId= handler that fans out these 4 queries.

-- name: GetCommentOrderStats :one
-- Counts by reservation status for the stats row (CommentOrderStatDTO).
-- total_detected = total reservations ever created for this post = keep codes
-- detected THAT HAD STOCK. Out-of-stock keep comments create no reservation and
-- are not counted here (N9) — the FE tile is labelled "ter-reserve", not "detected".
SELECT
    COUNT(*)                                                        AS total_detected,
    COUNT(*) FILTER (WHERE status = 'reserved')                    AS reserved_now,
    COUNT(*) FILTER (WHERE status = 'waiting-pay')                 AS waiting_pay,
    COUNT(*) FILTER (WHERE status = 'closed-wa')                   AS closed_wa,
    COUNT(*) FILTER (WHERE status = 'expired-released')            AS expired_released
FROM reservation
WHERE catalog_post_id = @catalog_post_id;

-- name: GetPostCommentCount :one
-- For postCommentsLabel in CommentOrderResponse.
SELECT comments_count FROM catalog_post WHERE id = @catalog_post_id;

-- name: ListRecentCommentsByPost :many
-- Incoming comments for the left panel (IncomingCommentDTO).
-- LEFT JOIN surfaces comments that were processed but did not match a code
-- (matched_code IS NULL, reservation_id IS NULL).
SELECT
    pc.ig_comment_id,
    pc.contact_handle,
    pc.comment_text,
    pc.received_at,
    r.code         AS matched_code,
    r.id           AS reservation_id,
    r.status       AS reservation_status
FROM processed_comment pc
LEFT JOIN reservation r ON r.ig_comment_id = pc.ig_comment_id
WHERE pc.account_id  = @account_id
  AND pc.ig_media_id = @ig_media_id
ORDER BY pc.received_at DESC
LIMIT @lim;

-- name: ListReservationsByPostWithProduct :many
-- Reservation rows with product context for the reservation panel (ReservationDTO).
SELECT
    r.id,
    r.code,
    r.contact_handle,
    r.contact_ig_user_id,
    r.status,
    r.expires_at,
    r.closed_at,
    r.wa_link,
    r.reserved_at,
    p.name        AS product_name,
    p.price_idr   AS product_price_idr
FROM reservation r
JOIN product p ON p.id = r.product_id
WHERE r.catalog_post_id = @catalog_post_id
ORDER BY r.reserved_at DESC
LIMIT @lim
OFFSET @off;

-- name: ListProductsByPost :many
-- Product list with live stock counters for the product panel (CatalogProductDTO).
SELECT
    code,
    name,
    price_idr,
    stock_left,
    stock_total
FROM product
WHERE catalog_post_id = @catalog_post_id
ORDER BY code;
