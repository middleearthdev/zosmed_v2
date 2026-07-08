-- +goose Up
-- Dedupe ledger untuk event messaging (mirror processed_comment 00004,
-- ADR-006 §5.2): Meta retry non-200 → cegah enqueue ganda. PK = ig message
-- id (message.mid); setiap subtype (DM/story-reply/story-mention/ad-referral)
-- membawa mid, jadi satu PK cukup — tidak perlu kolom subtype di key.
CREATE TABLE processed_message (
    ig_message_id      text        PRIMARY KEY,
    account_id         uuid        NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    subtype            text        NOT NULL DEFAULT 'dm',
    contact_ig_user_id text        NOT NULL DEFAULT '',
    received_at        timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS processed_message;
