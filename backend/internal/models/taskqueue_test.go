package models

import (
	"errors"
	"testing"
	"time"
)

// newTestQueue creates an isolated TaskQueue with its own worker goroutine.
// The queue is not drained on cleanup because there are no pending tasks
// by the time each test exits.
func newTestQueue() *TaskQueue {
	tq := &TaskQueue{tasks: make(chan Task, 10)}
	go tq.runWorker()
	return tq
}

// ---------------------------------------------------------------------------
// EnQueueReturn — blocking task with value and error
// ---------------------------------------------------------------------------

func TestTaskQueue_EnQueueReturn_ReturnsValue(t *testing.T) {
	tq := newTestQueue()

	val, err := tq.EnQueueReturn(func() (any, error) {
		return "result", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(string) != "result" {
		t.Errorf("val = %q, want %q", val, "result")
	}
}

func TestTaskQueue_EnQueueReturn_PropagatesError(t *testing.T) {
	tq := newTestQueue()
	sentinel := errors.New("task failed")

	_, err := tq.EnQueueReturn(func() (any, error) {
		return nil, sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

// ---------------------------------------------------------------------------
// EnQueue — fire-and-forget
// ---------------------------------------------------------------------------

func TestTaskQueue_EnQueue_RunsTask(t *testing.T) {
	tq := newTestQueue()
	done := make(chan struct{})

	tq.EnQueue(func() (any, error) {
		close(done)
		return nil, nil
	})

	select {
	case <-done:
		// task ran
	case <-time.After(time.Second):
		t.Error("fire-and-forget task did not run within 1s")
	}
}

// ---------------------------------------------------------------------------
// EnQueueReturnErrorOnlyTask — convenience wrapper
// ---------------------------------------------------------------------------

func TestTaskQueue_EnQueueReturnErrorOnlyTask_Nil(t *testing.T) {
	tq := newTestQueue()

	err := tq.EnQueueReturnErrorOnlyTask(func() error { return nil })
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTaskQueue_EnQueueReturnErrorOnlyTask_Error(t *testing.T) {
	tq := newTestQueue()
	sentinel := errors.New("oops")

	err := tq.EnQueueReturnErrorOnlyTask(func() error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

// ---------------------------------------------------------------------------
// EnQueueReturnValueOnlyTask — convenience wrapper
// ---------------------------------------------------------------------------

func TestTaskQueue_EnQueueReturnValueOnlyTask(t *testing.T) {
	tq := newTestQueue()

	val := tq.EnQueueReturnValueOnlyTask(func() any { return int64(42) })
	if val.(int64) != 42 {
		t.Errorf("val = %v, want 42", val)
	}
}

// ---------------------------------------------------------------------------
// Drain — flushes all previously enqueued work
// ---------------------------------------------------------------------------

func TestTaskQueue_Drain_WaitsForPendingTasks(t *testing.T) {
	tq := newTestQueue()

	counter := 0
	for range 5 {
		tq.EnQueue(func() (any, error) {
			counter++
			return nil, nil
		})
	}

	tq.Drain()

	if counter != 5 {
		t.Errorf("counter = %d after Drain, want 5", counter)
	}
}

// ---------------------------------------------------------------------------
// Serialization — tasks run in submission order
// ---------------------------------------------------------------------------

func TestTaskQueue_TasksRunInOrder(t *testing.T) {
	tq := newTestQueue()

	var order []int
	for i := range 5 {
		i := i
		tq.EnQueue(func() (any, error) {
			order = append(order, i)
			return nil, nil
		})
	}
	tq.Drain()

	for i, v := range order {
		if v != i {
			t.Errorf("order[%d] = %d, want %d (tasks ran out of order)", i, v, i)
		}
	}
}
