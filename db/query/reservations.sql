-- Reservation CRUD + state machine queries (ADR-001 §2).
-- Guard: UpdateReservationStatus uses WHERE status = @expected_status to prevent
-- lost-update races when multiple workers process the same reservation concurrently.

-- name: CreateReservation :one
INSERT INTO reservation (
    account_id,
    catalog_post_id,
    product_id,
    code,
    ig_comment_id,
    contact_ig_user_id,
    contact_handle,
    hold_seconds,
    expires_at,
    wa_link
) VALUES (
    @account_id,
    @catalog_post_id,
    @product_id,
    @code,
    @ig_comment_id,
    @contact_ig_user_id,
    @contact_handle,
    @hold_seconds,
    @expires_at,
    @wa_link
)
RETURNING *;

-- name: GetReservation :one
SELECT * FROM reservation WHERE id = @id;

-- name: UpdateReservationStatus :one
-- Race guard: only transitions from the expected non-terminal state succeed.
-- Caller passes closed_at = now() for terminal states (closed-wa, expired-released),
-- nil for active transitions (reserved → waiting-pay).
UPDATE reservation
SET
    status    = @new_status::reservation_status,
    closed_at = sqlc.narg('closed_at')
WHERE id     = @id
  AND status = @expected_status::reservation_status
RETURNING *;

-- name: ListReservationsByPost :many
SELECT * FROM reservation
WHERE catalog_post_id = @catalog_post_id
ORDER BY reserved_at DESC;

-- name: ListReservationsByAccount :many
SELECT * FROM reservation
WHERE account_id = @account_id
ORDER BY reserved_at DESC
LIMIT @lim
OFFSET @off;
