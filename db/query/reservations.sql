-- Reservation CRUD + state machine queries (ADR-001 §2).
-- Guard: UpdateReservationStatus uses WHERE status = @expected_status to prevent
-- lost-update races when multiple workers process the same reservation concurrently.

-- name: CreateReservation :one
-- Idempotent insert (ADR-007 §2.3b, #6b): ON CONFLICT (account_id, ig_comment_id)
-- DO NOTHING means a re-run (asynq retry) for a keep/C comment already reserved
-- returns ZERO rows -> pgx surfaces this as pgx.ErrNoRows ("no rows in result
-- set") from the :one QueryRow+Scan, exactly like GetProductByPostAndCode's
-- no-match case. Caller (libs/kits/seller ReservationService.Reserve) MUST
-- check isNoRows(err) and, on conflict, call GetReservationByComment to fetch
-- the existing reservation instead of treating this as a hard failure.
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
ON CONFLICT (account_id, ig_comment_id) DO NOTHING
RETURNING *;

-- name: GetReservation :one
SELECT * FROM reservation WHERE id = @id;

-- name: GetReservationByComment :one
-- Fetch the existing reservation for (account_id, ig_comment_id) when
-- CreateReservation hits the ON CONFLICT DO NOTHING branch above (ADR-007 #6b).
SELECT * FROM reservation
WHERE account_id = @account_id
  AND ig_comment_id = @ig_comment_id;

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

-- name: ListExpiredActiveReservations :many
-- Backstop sweep (MAJOR-3b): active reservations already past expiry, for the
-- reservation:reconcile task. Uses the partial index reservation_active_expires_at_idx.
SELECT id FROM reservation
WHERE status IN ('reserved', 'waiting-pay')
  AND expires_at < now()
ORDER BY expires_at
LIMIT @lim;
