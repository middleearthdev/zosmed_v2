-- user_session queries backing RequireUser (ADR-003 §2.3/§3). Only the
-- token_hash (SHA-256 of the raw cookie value) ever touches Postgres.

-- name: CreateSession :exec
INSERT INTO user_session (token_hash, user_id, expires_at) VALUES (@token_hash, @user_id, @expires_at);

-- name: GetSessionUser :one
-- Session lookup for RequireUser: token_hash -> user (join), rejecting expired sessions.
SELECT u.* FROM user_session s
JOIN app_user u ON u.id = s.user_id
WHERE s.token_hash = @token_hash AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM user_session WHERE token_hash = @token_hash;

-- name: DeleteExpiredSessions :exec
DELETE FROM user_session WHERE expires_at <= now();
