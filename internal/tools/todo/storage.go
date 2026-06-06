package todo

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"fmt"
	"sync"
)

// Store manages todos per session+agent scope.
type Store interface {
	Add(session.ID, agent.ID, []TodoItem)
	Update(session.ID, agent.ID, int, Status) error
	List(session.ID, agent.ID) []TodoItem
}

type TodoItem struct {
	ID     int
	Title  string
	Status Status
}

type storeKey struct {
	sessID  session.ID
	agentID agent.ID
}

var _ Store = (*InMemoryStore)(nil)

// InMemoryStore is a simple thread-safe in-memory Store.
type InMemoryStore struct {
	mu    sync.Mutex
	todos map[storeKey][]TodoItem
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		todos: map[storeKey][]TodoItem{},
	}
}

func (s *InMemoryStore) Add(sessionID session.ID, agentID agent.ID, items []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := storeKey{sessID: sessionID, agentID: agentID}
	for i, item := range items {
		item.ID = i
		item.Status = Pending
		s.todos[key] = append(s.todos[key], item)
	}
}

func (s *InMemoryStore) Update(sessionID session.ID, agentID agent.ID, todoID int, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := storeKey{sessID: sessionID, agentID: agentID}

	sessTodos := s.todos[key]

	if len(sessTodos) <= todoID || todoID < 0 {
		return fmt.Errorf("bad todo id")
	}

	s.todos[key][todoID].Status = status

	return nil
}

func (s *InMemoryStore) List(sessionID session.ID, agentID agent.ID) []TodoItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := storeKey{sessID: sessionID, agentID: agentID}

	items := make([]TodoItem, len(s.todos[key]))
	copy(items, s.todos[key])

	return items
}
