-- +goose Up
-- Workflow persistence (ADR-004 §2.1). Engine (libs/workflow) stays neutral —
-- this table only stores the graph metadata; compilation happens at runtime
-- (libs/workflow/compile.go).
CREATE TYPE workflow_status AS ENUM ('draft', 'live', 'paused', 'error');
-- Sinkron dengan packages/types/src/domain.ts WorkflowStatus.

CREATE TABLE workflow (
    id          uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid            NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    name        text            NOT NULL,
    status      workflow_status NOT NULL DEFAULT 'draft',
    segment     text            NOT NULL CHECK (segment IN ('seller','creator','booking')),
    -- segment hanya untuk pilihan preset/palette & AI persona; engine ABAIKAN ini (netral).
    version     int             NOT NULL DEFAULT 1,   -- bump saat publish; untuk cache-bust & run_log snapshot
    created_at  timestamptz     NOT NULL DEFAULT now(),
    updated_at  timestamptz     NOT NULL DEFAULT now()
);
CREATE INDEX workflow_account_id_idx ON workflow(account_id);
-- Hot path loader: hanya workflow live per akun.
CREATE INDEX workflow_live_idx ON workflow(account_id) WHERE status = 'live';

-- +goose Down
DROP INDEX IF EXISTS workflow_live_idx;
DROP INDEX IF EXISTS workflow_account_id_idx;
DROP TABLE IF EXISTS workflow;
DROP TYPE IF EXISTS workflow_status;
