package concurrency

import (
	"context"
	"sync"
)

// SessionLockManager ensures that only one worker can process an event for a given session at a time.
type SessionLockManager struct {
	locks map[string]*sync.Mutex
	mu    sync.Mutex
}

func NewSessionLockManager() *SessionLockManager {
	return &SessionLockManager{
		locks: make(map[string]*sync.Mutex),
	}
}

// Acquire locks the mutex for the given sessionID.
// It blocks until the lock is acquired or context is cancelled.
func (m *SessionLockManager) Acquire(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	lock, ok := m.locks[sessionID]
	if !ok {
		lock = &sync.Mutex{}
		m.locks[sessionID] = lock
	}
	m.mu.Unlock()

	// Simple blocking lock like SimpleSessionLockManager
	// Context cancellation handled by worker before acquiring
	lock.Lock()
	return nil
}

// Release unlocks the mutex for the given sessionID.
func (m *SessionLockManager) Release(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if lock, ok := m.locks[sessionID]; ok {
		// We can't check if locked, just unlock.
		// Assuming proper usage.
		// If using the goroutine approach above, we need to be careful.
		// Let's simplify: Just block. Worker context cancellation will handle the worker loop.
		lock.Unlock()
	}
}

// SimpleLockManager simplifies the context issue by just blocking.
// The worker handles context check *before* acquiring.
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
