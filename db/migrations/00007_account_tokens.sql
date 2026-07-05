-- +goose Up
-- Token storage for Instagram Login (Business Login for Instagram, CLAUDE.md
-- §4.0 / ADR-002 §4.2). access_token is an IG-user-scoped long-lived token —
-- there is deliberately NO "page_token"/"page_id" column; Facebook Page
-- tokens are out of scope entirely for Zosmed's Instagram integration.
ALTER TABLE account
    ADD COLUMN access_token       text        NOT NULL DEFAULT '',
    ADD COLUMN token_type         text        NOT NULL DEFAULT 'bearer',
    ADD COLUMN scopes             text[]      NOT NULL DEFAULT '{}',
    ADD COLUMN token_expires_at   timestamptz,
    ADD COLUMN token_refreshed_at timestamptz;

-- Supports the refresh-sweep hot path: find connected accounts whose token
-- is approaching expiry (ADR-002 §5).
CREATE INDEX account_refresh_due_idx ON account (token_expires_at)
    WHERE status = 'connected';

-- +goose Down
DROP INDEX account_refresh_due_idx;
ALTER TABLE account
    DROP COLUMN access_token,
    DROP COLUMN token_type,
    DROP COLUMN scopes,
    DROP COLUMN token_expires_at,
    DROP COLUMN token_refreshed_at;
