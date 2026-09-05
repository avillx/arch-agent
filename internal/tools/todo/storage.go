package todo

import (
	"arch-agent/internal/session"
	"fmt"
	"sync"
)

// Store manages todos per session+agent scope.
type Store interface {
	Add(session.ID, []TodoItem)
	Update(session.ID, int, Status) error
	List(session.ID) []TodoItem
}

type TodoItem struct {
	ID     int
	Title  string
	Status Status
}

func (i *TodoItem) String() string {
	badge := statusBadge[i.Status]
	return fmt.Sprintf("- %s #%d %s", badge, i.ID, i.Title)
}

var _ Store = (*InMemoryStore)(nil)

// InMemoryStore is a simple thread-safe in-memory Store.
type InMemoryStore struct {
	mu    sync.Mutex
	todos map[session.ID][]TodoItem
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		todos: map[session.ID][]TodoItem{},
	}
}

func (s *InMemoryStore) Add(sessID session.ID, items []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentLen := len(s.todos[sessID])
	for i, item := range items {
		item.ID = currentLen + i
		item.Status = Pending
		s.todos[sessID] = append(s.todos[sessID], item)
	}
}

func (s *InMemoryStore) Update(sessID session.ID, todoID int, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessTodos := s.todos[sessID]

	if len(sessTodos) <= todoID || todoID < 0 {
		return fmt.Errorf("bad todo id")
	}

	s.todos[sessID][todoID].Status = status

	return nil
}

func (s *InMemoryStore) List(sessID session.ID) []TodoItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]TodoItem, len(s.todos[sessID]))
	copy(items, s.todos[sessID])

	return items
}
