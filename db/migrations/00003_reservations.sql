-- +goose Up
-- Canonical status enum (CLAUDE.md §6 / §8.1 / ADR-001 §2).
-- Values MUST stay in sync with packages/types/src/domain.ts ReservationStatus
-- and packages/types/src/constants.ts RESERVATION_STATUSES.
CREATE TYPE reservation_status AS ENUM (
    'reserved',
    'waiting-pay',
    'closed-wa',
    'expired-released'
);

-- reservation: one row per keep/C event (ADR-001 §2 state machine).
-- hold_seconds stored per-row for audit consistency if default changes later.
CREATE TABLE reservation (
    id                   uuid               PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id           uuid               NOT NULL REFERENCES account(id),
    catalog_post_id      uuid               NOT NULL REFERENCES catalog_post(id),
    product_id           uuid               NOT NULL REFERENCES product(id),
    code                 text               NOT NULL,   -- e.g. "C1", matched code
    ig_comment_id        text               NOT NULL,   -- IG comment id that triggered
    contact_ig_user_id   text               NOT NULL,
    contact_handle       text               NOT NULL DEFAULT '',
    status               reservation_status NOT NULL DEFAULT 'reserved',
    hold_seconds         int                NOT NULL DEFAULT 300,
    reserved_at          timestamptz        NOT NULL DEFAULT now(),
    expires_at           timestamptz        NOT NULL,
    closed_at            timestamptz,                   -- NULL until terminal state
    wa_link              text               NOT NULL DEFAULT ''
);

CREATE INDEX reservation_account_id_idx       ON reservation(account_id);
CREATE INDEX reservation_catalog_post_id_idx  ON reservation(catalog_post_id);
CREATE INDEX reservation_ig_comment_id_idx    ON reservation(ig_comment_id);

-- Partial index for the expire worker hot-path: find active reservations past expiry.
CREATE INDEX reservation_active_expires_at_idx ON reservation(expires_at)
    WHERE status IN ('reserved', 'waiting-pay');

-- +goose Down
DROP TABLE reservation;
DROP TYPE reservation_status;
