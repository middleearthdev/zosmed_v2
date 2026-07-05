-- +goose Up
-- user_session: server-side session for app_user login (ADR-003 §2.2).
-- Only the SHA-256 hash of the raw token is stored — the raw token lives only
-- in the httpOnly "zsid" cookie, never in Postgres (AC-2).
CREATE TABLE user_session (
    token_hash text        PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX user_session_user_idx ON user_session (user_id);   -- logout-all / cleanup per user

-- +goose Down
DROP TABLE user_session;
