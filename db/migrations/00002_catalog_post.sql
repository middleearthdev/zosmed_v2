-- +goose Up
-- catalog_post: post/Reel registered by the seller for comment-to-order (§8.1.4).
-- Only posts in this table are eligible for keep/C detection.
CREATE TABLE catalog_post (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      uuid        NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    ig_media_id     text        NOT NULL,
    caption         text        NOT NULL DEFAULT '',
    comments_count  int         NOT NULL DEFAULT 0,
    active          bool        NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now(),

    UNIQUE (account_id, ig_media_id)
);

CREATE INDEX catalog_post_account_id_idx   ON catalog_post(account_id);
-- Partial index used by GetActiveCatalogPostByMedia (ingest fast-path).
CREATE INDEX catalog_post_media_active_idx ON catalog_post(ig_media_id, account_id) WHERE active = true;

-- product: one row per purchasable item on a catalog post; code = the keep/C label.
-- stock_left is the live inventory counter; stock_total is the ceiling.
CREATE TABLE product (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_post_id uuid        NOT NULL REFERENCES catalog_post(id) ON DELETE CASCADE,
    code            text        NOT NULL,           -- e.g. "C1", "C3", "keep"
    name            text        NOT NULL,
    price_idr       bigint      NOT NULL DEFAULT 0,
    stock_total     int         NOT NULL DEFAULT 0,
    stock_left      int         NOT NULL DEFAULT 0,

    UNIQUE (catalog_post_id, code),
    CHECK (stock_left >= 0),
    CHECK (stock_total >= 0),
    CHECK (stock_left <= stock_total)
);

CREATE INDEX product_catalog_post_id_idx ON product(catalog_post_id);

-- +goose Down
DROP TABLE product;
DROP TABLE catalog_post;
