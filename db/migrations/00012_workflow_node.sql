-- +goose Up
-- Workflow node persistence (ADR-004 §2.2). node_type is NOT DB-CHECK'd — the
-- feasible catalog (§7) evolves in code (libs/workflow/nodes/catalog.go); the
-- app layer validates node_type against that single source at save/activate
-- time so the guardrail (CLAUDE.md §4b) is enforced without a migration per
-- new node type.
CREATE TABLE workflow_node (
    id           uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id  uuid  NOT NULL REFERENCES workflow(id) ON DELETE CASCADE,
    category     text  NOT NULL CHECK (category IN ('trigger','filter','action')),
    node_type    text  NOT NULL,
    config       jsonb NOT NULL DEFAULT '{}'::jsonb,
    position_x   int   NOT NULL DEFAULT 0,
    position_y   int   NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workflow_node_workflow_id_idx ON workflow_node(workflow_id);

-- +goose Down
DROP INDEX IF EXISTS workflow_node_workflow_id_idx;
DROP TABLE IF EXISTS workflow_node;
