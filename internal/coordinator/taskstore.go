package coordinator

import (
	"sync"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

const (
	taskTTL          = 1 * time.Hour
	taskCleanupEvery = 5 * time.Minute
)

// TaskStore is an in-memory store for async tasks with TTL-based cleanup.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*types.Task
	stop  chan struct{}
}

// NewTaskStore creates a TaskStore and starts the background cleanup goroutine.
func NewTaskStore() *TaskStore {
	ts := &TaskStore{
		tasks: make(map[string]*types.Task),
		stop:  make(chan struct{}),
	}
	go ts.cleanupLoop()
	return ts
}

// Create adds a new task to the store.
func (ts *TaskStore) Create(task *types.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tasks[task.ID] = task
}

// Get returns a task by ID, or nil if not found.
func (ts *TaskStore) Get(id string) *types.Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	t := ts.tasks[id]
	if t == nil {
		return nil
	}
	// Return a copy to avoid races.
	cp := *t
	return &cp
}

// UpdateStatus sets the task's status, final response, and error.
func (ts *TaskStore) UpdateStatus(id string, status types.TaskStatus, response, errMsg string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t := ts.tasks[id]
	if t == nil {
		return
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	if response != "" {
		t.Response = response
	}
	if errMsg != "" {
		t.Error = errMsg
	}
}

// UpdatePartial sets the task's partial response for streaming.
func (ts *TaskStore) UpdatePartial(id string, partial string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t := ts.tasks[id]
	if t == nil {
		return
	}
	t.PartialResponse = partial
	t.UpdatedAt = time.Now()
}

// Stop terminates the background cleanup goroutine.
func (ts *TaskStore) Stop() {
	close(ts.stop)
}

func (ts *TaskStore) cleanupLoop() {
	ticker := time.NewTicker(taskCleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ts.stop:
			return
		case <-ticker.C:
			ts.evict()
		}
	}
}

func (ts *TaskStore) evict() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	cutoff := time.Now().Add(-taskTTL)
	for id, t := range ts.tasks {
		if t.UpdatedAt.Before(cutoff) {
			delete(ts.tasks, id)
		}
	}
}
