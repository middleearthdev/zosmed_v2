-- +goose Up
-- Link a connected Instagram account to the Zosmed user who owns it
-- (ADR-003 §2.2). Nullable: accounts can exist before any user links them
-- (e.g. legacy rows, seed order); the connect callback fills user_id from the
-- verified signed state. MVP is 1 user <-> 0..1 account, enforced in
-- application code (GetAccountByUserID takes one) rather than a DB
-- constraint — multi-account per user is a later phase (ADR-003 §2.2 note).
ALTER TABLE account
    ADD COLUMN user_id uuid REFERENCES app_user(id) ON DELETE SET NULL;
CREATE INDEX account_user_idx ON account (user_id);

-- +goose Down
DROP INDEX account_user_idx;
ALTER TABLE account DROP COLUMN user_id;
