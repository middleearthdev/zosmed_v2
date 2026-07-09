package enqueue

// enqueue_test.go — unit test for isDuplicateTaskID (ADR-007 §2.2/§3.5): the
// translation from asynq's "already enqueued under this TaskID" errors into
// success. This is the mechanism that makes EnqueueCommentIngest/
// EnqueueDMIngest idempotent — a re-delivered webhook event that reaches
// enqueue twice must never produce a second asynq task.
//
// EnqueueCommentIngest/EnqueueDMIngest themselves are exercised indirectly
// via apps/api/internal/webhook's handler tests (fakeEnqueuer, ADR-007 §5
// scenarios 6–8) — those assert the caller-side contract (error vs nil)
// without needing a real Redis/asynq.Client. Testing the full asynq.Client
// round-trip (a genuine TaskID conflict) would require a Redis test double
// (e.g. miniredis) not currently a dependency of apps/api (§12a-4 — not
// worth adding for one assertion asynq's own test suite already covers).

import (
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestIsDuplicateTaskID_TaskIDConflict_TreatedAsDuplicate(t *testing.T) {
	if !isDuplicateTaskID(asynq.ErrTaskIDConflict) {
		t.Error("expected ErrTaskIDConflict to be treated as a duplicate (already durably enqueued)")
	}
}

func TestIsDuplicateTaskID_WrappedTaskIDConflict_TreatedAsDuplicate(t *testing.T) {
	wrapped := errors.Join(errors.New("enqueue: comment ingest task"), asynq.ErrTaskIDConflict)
	if !isDuplicateTaskID(wrapped) {
		t.Error("expected a wrapped ErrTaskIDConflict to still be detected via errors.Is")
	}
}

func TestIsDuplicateTaskID_ErrDuplicateTask_TreatedAsDuplicate(t *testing.T) {
	if !isDuplicateTaskID(asynq.ErrDuplicateTask) {
		t.Error("expected ErrDuplicateTask to be treated as a duplicate (belt-and-suspenders)")
	}
}

func TestIsDuplicateTaskID_OtherError_NotDuplicate(t *testing.T) {
	if isDuplicateTaskID(errors.New("redis: connection refused")) {
		t.Error("expected a genuine infra error to NOT be treated as a duplicate")
	}
}

func TestIsDuplicateTaskID_Nil_NotDuplicate(t *testing.T) {
	if isDuplicateTaskID(nil) {
		t.Error("expected nil error to not be treated as a duplicate")
	}
}
