-- Account + token store queries (ADR-002 §4.3). Token is the sole credential
-- source per account — connect flow and refresh scheduler never persist it
-- anywhere else (DRY §12a-1).

-- name: GetAccountByIgUserID :one
-- Webhook resolution: entry.id (IGSID) → account row + token (ADR-002 §6.1).
SELECT * FROM account WHERE ig_user_id = @ig_user_id;

-- name: GetAccountByID :one
-- Worker resolution: account_id (UUID, from task payload) → token + ig_user_id (ADR-002 §6.2).
SELECT * FROM account WHERE id = @id;

-- name: UpsertAccountFromOAuth :one
-- Connect callback: persist a freshly (re-)connected account. Re-connecting
-- an existing ig_user_id refreshes its token and clears any prior 'expired' status.
INSERT INTO account (
    ig_user_id,
    handle,
    display_name,
    access_token,
    token_type,
    scopes,
    token_expires_at,
    token_refreshed_at,
    status
) VALUES (
    @ig_user_id,
    @handle,
    @display_name,
    @access_token,
    @token_type,
    @scopes,
    @token_expires_at,
    now(),
    'connected'
)
ON CONFLICT (ig_user_id) DO UPDATE SET
    handle             = EXCLUDED.handle,
    display_name       = EXCLUDED.display_name,
    access_token       = EXCLUDED.access_token,
    token_type         = EXCLUDED.token_type,
    scopes             = EXCLUDED.scopes,
    token_expires_at   = EXCLUDED.token_expires_at,
    token_refreshed_at = now(),
    status             = 'connected'
RETURNING *;

-- name: UpdateAccountToken :exec
-- Refresh scheduler: persist a successfully refreshed token (ADR-002 §5 step 2).
UPDATE account
SET access_token       = @access_token,
    token_expires_at   = @token_expires_at,
    token_refreshed_at = @token_refreshed_at
WHERE id = @id;

-- name: MarkAccountExpired :exec
-- Refresh scheduler: refresh failed — mark the account expired rather than
-- keep sending with a dead token (ADR-002 §5 step 2).
UPDATE account SET status = 'expired' WHERE id = @id;

-- name: ListAccountsDueForRefresh :many
-- Refresh scheduler sweep: connected accounts whose token expires before
-- @threshold (now + lead time, ADR-002 §5 step 1).
SELECT * FROM account
WHERE status = 'connected'
  AND token_expires_at IS NOT NULL
  AND token_expires_at < @threshold
ORDER BY token_expires_at ASC;
