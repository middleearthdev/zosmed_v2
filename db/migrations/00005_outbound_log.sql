-- +goose Up
-- outbound_log: audit trail for every outbound IG message (§10 safety audit).
-- Safety layer (libs/safety) writes here after each successful outbound.
-- kind values: 'private-reply' | 'dm' | 'comment-reply'  (open-ended text, not enum,
-- to allow extension without migration).
CREATE TABLE outbound_log (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      uuid        NOT NULL REFERENCES account(id),
    kind            text        NOT NULL,   -- 'private-reply' | 'dm' | 'comment-reply'
    target_user_id  text        NOT NULL,
    trigger_key     text        NOT NULL,   -- (account_id, user_id, trigger) dedupe key
    ig_object_id    text        NOT NULL DEFAULT '',  -- comment_id or media_id
    sent_at         timestamptz NOT NULL DEFAULT now()
);

-- Index for rate-limit counting by safety layer (account + kind + recent window).
CREATE INDEX outbound_log_account_kind_sent_idx ON outbound_log(account_id, kind, sent_at DESC);

-- +goose Down
DROP TABLE outbound_log;
