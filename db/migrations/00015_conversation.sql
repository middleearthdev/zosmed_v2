-- +goose Up
-- Window/last-interaction store (§4c 24h, ADR-006 §5.1). Satu baris per
-- (akun, kontak). last_interaction_at di-refresh setiap event MESSAGING masuk
-- (DM/story-reply/story-mention/ad-referral) — semuanya membuka window
-- messaging (§4c, ADR-006 R3). Komentar TIDAK membuka window messaging →
-- tidak menyentuh tabel ini (comment_ingest.go tak diubah).
CREATE TABLE conversation (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id          uuid        NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    contact_ig_user_id  text        NOT NULL,                 -- IGSID kontak (§4.0)
    last_interaction_at timestamptz NOT NULL,                 -- sumber window 24h
    last_source         text        NOT NULL DEFAULT 'dm',    -- 'dm' (semua permukaan messaging)
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, contact_ig_user_id)
);
CREATE INDEX conversation_account_contact_idx ON conversation(account_id, contact_ig_user_id);

-- +goose Down
DROP INDEX IF EXISTS conversation_account_contact_idx;
DROP TABLE IF EXISTS conversation;
