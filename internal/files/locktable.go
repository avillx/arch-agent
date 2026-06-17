package files

import "sync"

type entry struct {
	mu   sync.RWMutex
	refs int
}

type lockTable struct {
	mu    sync.Mutex
	locks map[string]*entry
}

func newLockTable() *lockTable {
	return &lockTable{locks: map[string]*entry{}}
}

func (lt *lockTable) acquire(path string) *entry {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	e := lt.locks[path]
	if e == nil {
		e = &entry{}
		lt.locks[path] = e
	}
	e.refs++
	return e
}

func (lt *lockTable) release(path string, e *entry) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	e.refs--
	if e.refs == 0 {
		delete(lt.locks, path)
	}
}

func (lt *lockTable) Lock(path string) func() {
	e := lt.acquire(path)
	e.mu.Lock()
	return func() {
		e.mu.Unlock()
		lt.release(path, e)
	}
}

func (lt *lockTable) RLock(path string) func() {
	e := lt.acquire(path)
	e.mu.RLock()
	return func() {
		e.mu.RUnlock()
		lt.release(path, e)
	}
}
