-- +goose Up
-- Workflow edge persistence (ADR-004 §2.3). Edges connect action nodes for
-- ordering purposes (compiler §4.3 topo-sorts ActionKeys from these edges);
-- edges between other categories are tolerated but currently unused by the
-- compiler (trigger/filter ordering is OR/AND, not sequence-sensitive).
CREATE TABLE workflow_edge (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id  uuid NOT NULL REFERENCES workflow(id) ON DELETE CASCADE,
    from_node_id uuid NOT NULL REFERENCES workflow_node(id) ON DELETE CASCADE,
    to_node_id   uuid NOT NULL REFERENCES workflow_node(id) ON DELETE CASCADE,
    UNIQUE (workflow_id, from_node_id, to_node_id)
);
CREATE INDEX workflow_edge_workflow_id_idx ON workflow_edge(workflow_id);

-- +goose Down
DROP INDEX IF EXISTS workflow_edge_workflow_id_idx;
DROP TABLE IF EXISTS workflow_edge;
