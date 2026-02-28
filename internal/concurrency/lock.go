package concurrency

import "sync"

// SimpleSessionLockManager serializes per-session processing.
type SimpleSessionLockManager struct {
	locks map[string]*sync.Mutex
	mu    sync.Mutex
}

func NewSimpleSessionLockManager() *SimpleSessionLockManager {
	return &SimpleSessionLockManager{
		locks: make(map[string]*sync.Mutex),
	}
}

func (m *SimpleSessionLockManager) Lock(sessionID string) {
	m.mu.Lock()
	lock, ok := m.locks[sessionID]
	if !ok {
		lock = &sync.Mutex{}
		m.locks[sessionID] = lock
	}
	m.mu.Unlock()
	lock.Lock()
}

func (m *SimpleSessionLockManager) Unlock(sessionID string) {
	m.mu.Lock()
	lock, ok := m.locks[sessionID]
	if ok {
		lock.Unlock()
	}
	m.mu.Unlock()
}
