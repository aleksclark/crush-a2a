package bridge

import (
	"sync"

	"github.com/aleksclark/crush-a2a/internal/a2a"
)

// TaskEntry holds the mapping between an A2A task and its Crush workspace/session.
type TaskEntry struct {
	TaskID      string
	ContextID   string
	WorkspaceID string
	SessionID   string
	Task        *a2a.Task
}

// TaskStore is a concurrent-safe in-memory store for task state.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*TaskEntry
}

// NewTaskStore creates a new task store.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*TaskEntry),
	}
}

// Put stores or updates a task entry.
func (s *TaskStore) Put(entry *TaskEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[entry.TaskID] = entry
}

// Get retrieves a task entry by A2A task ID.
func (s *TaskStore) Get(taskID string) (*TaskEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.tasks[taskID]
	return e, ok
}

// List returns all task entries.
func (s *TaskStore) List() []*TaskEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]*TaskEntry, 0, len(s.tasks))
	for _, e := range s.tasks {
		entries = append(entries, e)
	}
	return entries
}
