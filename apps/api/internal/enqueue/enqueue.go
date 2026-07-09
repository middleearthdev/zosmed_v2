// Package enqueue wraps *asynq.Client to provide domain-level task enqueue methods.
// Transport handlers call these methods; they never construct asynq tasks directly (SoC §12a-3).
package enqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/zosmed/zosmed/libs/platform/tasks"
)

// ingestTaskRetention is how long a completed ingest task is kept in asynq's
// completed set (ADR-007 §2.2/§3.5). It bounds the window in which a
// re-delivered Meta webhook event is caught by asynq's own TaskID conflict
// detection, on top of the longer-lived ExistsProcessed* ledger read-check.
const ingestTaskRetention = 24 * time.Hour

// Enqueuer is the interface satisfied by *Client (and fakes in tests — see
// ADR-007 Tahap D, closes BUG-001 in handler_test.go). Extracted so the
// webhook handler's enqueue-first ordering (§2.2) can be unit-tested without
// a real asynq.Client/Redis.
type Enqueuer interface {
	EnqueueCommentIngest(ctx context.Context, p tasks.CommentIngestPayload) error
	EnqueueDMIngest(ctx context.Context, p tasks.DMIngestPayload) error
}

// Client wraps asynq.Client and exposes domain-level enqueue methods.
type Client struct {
	c *asynq.Client
}

// New creates a Client backed by the given asynq.Client.
func New(c *asynq.Client) *Client {
	return &Client{c: c}
}

var _ Enqueuer = (*Client)(nil)

// EnqueueCommentIngest enqueues a TaskCommentIngest for the worker to process.
// Called by the webhook handler BEFORE the processed_comment ledger write
// (ADR-007 §2.2 — enqueue-first ordering). Uses asynq.TaskID(comment_id) +
// Retention so a re-delivered event that reaches this call twice is
// idempotent at the asynq layer: a TaskID conflict is treated as success
// (the task is already durably enqueued), never as an error.
func (e *Client) EnqueueCommentIngest(ctx context.Context, p tasks.CommentIngestPayload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("enqueue: marshal comment ingest: %w", err)
	}
	_, err = e.c.EnqueueContext(ctx, asynq.NewTask(tasks.TaskCommentIngest, b),
		asynq.TaskID("ingest:comment:"+p.CommentID),
		asynq.Retention(ingestTaskRetention),
	)
	if isDuplicateTaskID(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue: comment ingest task: %w", err)
	}
	return nil
}

// EnqueueDMIngest enqueues a TaskDMIngest for the worker to process (ADR-006
// §3.3 step 4). Called by the webhook handler after account resolution,
// dedupe read-check, and the HasLiveWorkflow gate — but BEFORE the
// processed_message ledger write (ADR-007 §2.2). Same idempotent-enqueue
// treatment as EnqueueCommentIngest.
func (e *Client) EnqueueDMIngest(ctx context.Context, p tasks.DMIngestPayload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("enqueue: marshal dm ingest: %w", err)
	}
	_, err = e.c.EnqueueContext(ctx, asynq.NewTask(tasks.TaskDMIngest, b),
		asynq.TaskID("ingest:dm:"+p.MessageID),
		asynq.Retention(ingestTaskRetention),
	)
	if isDuplicateTaskID(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue: dm ingest task: %w", err)
	}
	return nil
}

// isDuplicateTaskID reports whether err indicates the task was already
// enqueued under the same TaskID — i.e. a prior enqueue attempt already
// succeeded durably, so this call should be treated as success (ADR-007
// §2.2/§3.5). asynq.TaskID conflicts surface as ErrTaskIDConflict; the
// Unique-option error ErrDuplicateTask is also treated as duplicate here in
// case the enqueue option ever changes to Unique (belt-and-suspenders, same
// "already enqueued" semantics from the caller's point of view).
func isDuplicateTaskID(err error) bool {
	return errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask)
}
