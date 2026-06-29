-- +goose Up
-- comment_order_settings: per-account keyword & hold config for Seller Kit (ADR-001 §4.4).
-- Default keywords from KIT_KEYWORDS.seller (packages/types/src/constants.ts) — keep in sync.
-- hold_seconds default = 300 (5 min), matching state machine (ADR-001 §2).
CREATE TABLE comment_order_settings (
    account_id      uuid     PRIMARY KEY REFERENCES account(id) ON DELETE CASCADE,
    keywords        text[]   NOT NULL DEFAULT ARRAY['keep','c','c1','c3','order'],
    hold_seconds    int      NOT NULL DEFAULT 300,
    reply_template  text     NOT NULL DEFAULT
        'Halo kak! Pesanan {{kode}} untuk {{produk}} sudah kami catat 🎉 Closing via WhatsApp ya: {{wa_link}}'
);

-- +goose Down
DROP TABLE comment_order_settings;
