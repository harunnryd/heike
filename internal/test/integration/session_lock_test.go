package integration_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/concurrency"
)

func TestSessionLockSingleflight(t *testing.T) {
	lockMgr := concurrency.NewSimpleSessionLockManager()
	sessionID := "test-session-singleflight"

	var counter int32
	var wg sync.WaitGroup

	numGoroutines := 10
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			lockMgr.Lock(sessionID)
			defer lockMgr.Unlock(sessionID)

			current := atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)

			if current != 1 {
				t.Errorf("Goroutine %d: Counter should be 1 during lock, got %d", id, current)
			}

			atomic.AddInt32(&counter, -1)
		}(i)
	}

	wg.Wait()

	finalCounter := atomic.LoadInt32(&counter)
	if finalCounter != 0 {
		t.Errorf("Final counter should be 0, got %d", finalCounter)
	}
}

func TestSessionLockDifferentSessions(t *testing.T) {
	lockMgr := concurrency.NewSimpleSessionLockManager()

	session1 := "session-1"
	session2 := "session-2"

	var wg sync.WaitGroup
	wg.Add(2)

	start := time.Now()

	go func() {
		lockMgr.Lock(session1)
		defer lockMgr.Unlock(session1)
		time.Sleep(50 * time.Millisecond)
		wg.Done()
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		lockMgr.Lock(session2)
		defer lockMgr.Unlock(session2)
		wg.Done()
	}()

	wg.Wait()

	elapsed := time.Since(start)
	if elapsed > 60*time.Millisecond {
		t.Errorf("Expected completion in ~50ms, took %v", elapsed)
	}
	if elapsed < 10*time.Millisecond {
		t.Errorf("Expected completion in at least 10ms, took %v", elapsed)
	}
}

func TestSessionLockConcurrency(t *testing.T) {
	lockMgr := concurrency.NewSimpleSessionLockManager()
	sessionID := "test-session-concurrency"

	var order []int
	var mu sync.Mutex

	numWorkers := 5
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()

			lockMgr.Lock(sessionID)
			defer lockMgr.Unlock(sessionID)

			mu.Lock()
			order = append(order, id)
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)
		}(i)
	}

	wg.Wait()

	if len(order) != numWorkers {
		t.Errorf("Expected %d executions, got %d", numWorkers, len(order))
	}

	seen := make(map[int]bool)
	for _, id := range order {
		if seen[id] {
			t.Errorf("Worker %d executed twice", id)
		}
		seen[id] = true
	}
}

func TestSessionLockStress(t *testing.T) {
	lockMgr := concurrency.NewSimpleSessionLockManager()
	sessionID := "test-session-stress"

	var counter int32
	var wg sync.WaitGroup

	numIterations := 100
	numGoroutines := 10

	wg.Add(numGoroutines * numIterations)

	for g := 0; g < numGoroutines; g++ {
		for i := 0; i < numIterations; i++ {
			go func() {
				defer wg.Done()

				lockMgr.Lock(sessionID)
				defer lockMgr.Unlock(sessionID)

				atomic.AddInt32(&counter, 1)
			}()
		}
	}

	wg.Wait()

	finalCounter := atomic.LoadInt32(&counter)
	expected := int32(numGoroutines * numIterations)
	if finalCounter != expected {
		t.Errorf("Expected counter %d, got %d", expected, finalCounter)
	}
}
