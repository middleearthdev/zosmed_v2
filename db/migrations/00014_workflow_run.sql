-- +goose Up
-- Workflow run log (ADR-004 §2.4). Written by the worker after Engine.Run
-- (libs/workflow/engine.go — UNCHANGED) so the Runs screen has an audit trail.
-- workflow_id is ON DELETE SET NULL so history survives workflow deletion;
-- workflow_name is snapshotted at write time for display (ADR-004 R4 — save
-- replaces workflow_node/workflow_edge rows, so node ids are not stable
-- across saves; we snapshot the workflow name instead of a node reference).
CREATE TABLE workflow_run (
    id             uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id    uuid  REFERENCES workflow(id) ON DELETE SET NULL,
    workflow_name  text  NOT NULL DEFAULT '',
    account_id     uuid  NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    trigger_source text  NOT NULL DEFAULT '',   -- 'comment' | 'dm' | 'story'
    trigger_summary text NOT NULL DEFAULT '',   -- mis. "comment by @rina_susanti"
    object_id      text  NOT NULL DEFAULT '',   -- ig comment/message id (dedupe/trace)
    status         text  NOT NULL CHECK (status IN ('success','failed','skipped')),
    triggered      bool  NOT NULL DEFAULT false,
    filter_passed  bool  NOT NULL DEFAULT false,
    steps          jsonb NOT NULL DEFAULT '[]'::jsonb, -- serialisasi []workflow.StepLog
    error          text  NOT NULL DEFAULT '',
    duration_ms    int   NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workflow_run_account_created_idx ON workflow_run(account_id, created_at DESC);
CREATE INDEX workflow_run_workflow_created_idx ON workflow_run(workflow_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS workflow_run_workflow_created_idx;
DROP INDEX IF EXISTS workflow_run_account_created_idx;
DROP TABLE IF EXISTS workflow_run;
