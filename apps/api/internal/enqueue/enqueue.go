// Package enqueue wraps *asynq.Client to provide domain-level task enqueue methods.
// Transport handlers call these methods; they never construct asynq tasks directly (SoC §12a-3).
package enqueue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/zosmed/zosmed/libs/platform/tasks"
)

// Client wraps asynq.Client and exposes domain-level enqueue methods.
type Client struct {
	c *asynq.Client
}

// New creates a Client backed by the given asynq.Client.
func New(c *asynq.Client) *Client {
	return &Client{c: c}
}

// EnqueueCommentIngest enqueues a TaskCommentIngest for the worker to process.
// Called by the webhook handler after signature verification and dedupe check.
func (e *Client) EnqueueCommentIngest(ctx context.Context, p tasks.CommentIngestPayload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("enqueue: marshal comment ingest: %w", err)
	}
	_, err = e.c.EnqueueContext(ctx, asynq.NewTask(tasks.TaskCommentIngest, b))
	if err != nil {
		return fmt.Errorf("enqueue: comment ingest task: %w", err)
	}
	return nil
}
