-- +goose Up
-- Minimal account table: IG Business/Creator account connected via OAuth (CLAUDE.md §6).
-- Enough for FK anchor in this slice; auth fields added by platform engineer.
CREATE TABLE account (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    ig_user_id   text        NOT NULL UNIQUE,
    handle       text        NOT NULL,
    display_name text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'connected'
                                CHECK (status IN ('connected', 'expired', 'disconnected')),
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE account;
