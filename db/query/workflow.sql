-- Workflow persistence queries (ADR-004 §2/§3). Scoping is always by
-- account_id in addition to id so a guessed workflow id from another account
-- can never be read/mutated (no separate authorization check needed in the
-- handler beyond resolving the caller's own account_id).

-- name: CreateWorkflow :one
INSERT INTO workflow (account_id, name, segment)
VALUES (@account_id, @name, @segment)
RETURNING *;

-- name: GetWorkflowByID :one
SELECT * FROM workflow WHERE id = @id AND account_id = @account_id;

-- name: ListWorkflowsByAccount :many
-- node_count powers the WorkflowSummary DTO without a second round-trip per row.
SELECT w.*, COUNT(n.id)::int AS node_count
FROM workflow w
LEFT JOIN workflow_node n ON n.workflow_id = w.id
WHERE w.account_id = @account_id
GROUP BY w.id
ORDER BY w.updated_at DESC;

-- name: ListLiveWorkflowsByAccount :many
-- Loader hot-path (ADR-004 §1): only 'live' workflows are compiled per event.
SELECT * FROM workflow
WHERE account_id = @account_id AND status = 'live'
ORDER BY created_at ASC;

-- name: HasLiveWorkflow :one
-- Ingest decoupling enabler (ADR-005 §3/B1): cheap existence check used by
-- apps/api/internal/webhook.processComment to decide whether a comment on a
-- non-catalog post should still be enqueued because the account has at least
-- one generic `live` workflow. Backed by the partial index workflow_live_idx
-- (WHERE status = 'live'), so this is an index-only existence probe, not a
-- full table scan.
SELECT EXISTS (
    SELECT 1 FROM workflow WHERE account_id = @account_id AND status = 'live'
) AS has_live;

-- name: UpdateWorkflowMeta :one
-- Save draft (PUT): renames the workflow and bumps updated_at. version is
-- intentionally NOT bumped here — version only bumps on activate (§2.1 comment).
UPDATE workflow
SET name = @name,
    updated_at = now()
WHERE id = @id AND account_id = @account_id
RETURNING *;

-- name: SetWorkflowStatus :one
-- Generic status transition (pause -> 'paused', or marking 'error'). Does not
-- bump version — only ActivateWorkflow does, since only publish needs cache-busting.
UPDATE workflow
SET status = @status::workflow_status,
    updated_at = now()
WHERE id = @id AND account_id = @account_id
RETURNING *;

-- name: ActivateWorkflow :one
-- Publish: status -> 'live' and version bumped (cache-bust + run_log snapshot, §2.1).
UPDATE workflow
SET status = 'live',
    version = version + 1,
    updated_at = now()
WHERE id = @id AND account_id = @account_id
RETURNING *;

-- name: DeleteWorkflow :execrows
DELETE FROM workflow WHERE id = @id AND account_id = @account_id;

-- name: ListNodesByWorkflow :many
SELECT * FROM workflow_node
WHERE workflow_id = @workflow_id
ORDER BY position_x ASC, created_at ASC;

-- name: ListEdgesByWorkflow :many
SELECT * FROM workflow_edge WHERE workflow_id = @workflow_id;

-- name: InsertNode :one
INSERT INTO workflow_node (workflow_id, category, node_type, config, position_x, position_y)
VALUES (@workflow_id, @category, @node_type, @config, @position_x, @position_y)
RETURNING *;

-- name: InsertEdge :one
INSERT INTO workflow_edge (workflow_id, from_node_id, to_node_id)
VALUES (@workflow_id, @from_node_id, @to_node_id)
RETURNING *;

-- name: DeleteNodesByWorkflow :exec
-- Cascades to workflow_edge via FK ON DELETE CASCADE; DeleteEdgesByWorkflow is
-- called first by the store regardless, so ordering here is not load-bearing.
DELETE FROM workflow_node WHERE workflow_id = @workflow_id;

-- name: DeleteEdgesByWorkflow :exec
DELETE FROM workflow_edge WHERE workflow_id = @workflow_id;

-- name: InsertRun :one
-- workflow_id is nullable (sqlc.narg): the transitional fallback built-in
-- comment-to-order workflow (ADR-004 R3) has no persisted workflow row.
INSERT INTO workflow_run (
    workflow_id, workflow_name, account_id, trigger_source, trigger_summary,
    object_id, status, triggered, filter_passed, steps, error, duration_ms
) VALUES (
    sqlc.narg('workflow_id'), @workflow_name, @account_id, @trigger_source, @trigger_summary,
    @object_id, @status, @triggered, @filter_passed, @steps, @error, @duration_ms
)
RETURNING *;

-- name: ListRunsByWorkflow :many
SELECT * FROM workflow_run
WHERE workflow_id = @workflow_id AND account_id = @account_id
ORDER BY created_at DESC
LIMIT @lim;

-- name: ListRunsByAccount :many
SELECT * FROM workflow_run
WHERE account_id = @account_id
ORDER BY created_at DESC
LIMIT @lim;
