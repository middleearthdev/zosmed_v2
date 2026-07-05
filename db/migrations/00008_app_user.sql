-- +goose Up
-- app_user: Zosmed's own login identity (ADR-003 §2.2), separate from the
-- Instagram `account` table. Table name avoids the reserved word "user".
CREATE TABLE app_user (
    id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   text        NOT NULL UNIQUE,       -- stored lower-case (normalised in app)
    password_hash           text        NOT NULL,              -- bcrypt; never sent to FE
    segment                 text            CHECK (segment IN ('seller','creator','booking')), -- NULL until onboarding
    onboarding_completed_at timestamptz,                       -- NULL = onboarding not finished
    created_at              timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE app_user;
