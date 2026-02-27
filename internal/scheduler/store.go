package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/natefinch/atomic"
	"github.com/oklog/ulid/v2"
	"github.com/robfig/cron/v3"
)

type LeaseStatus string

const (
	StatusIdle   LeaseStatus = "IDLE"
	StatusLeased LeaseStatus = "LEASED"
	StatusDone   LeaseStatus = "DONE"
	StatusFailed LeaseStatus = "FAILED"
)

type Lease struct {
	RunID     string      `json:"run_id"`
	Status    LeaseStatus `json:"status"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type Task struct {
	ID          string    `json:"id"`
	Schedule    string    `json:"schedule"` // Cron spec or "@every 1h"
	Description string    `json:"description"`
	NextRun     time.Time `json:"next_run"`
	Lease       *Lease    `json:"lease,omitempty"`
	Content     string    `json:"content,omitempty"` // Task content to execute
}

type TaskList struct {
	Tasks map[string]*Task `json:"tasks"`
}

type Store struct {
	path string
	data TaskList
	mu   sync.RWMutex
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path: path,
		data: TaskList{Tasks: make(map[string]*Task)},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return nil
	}
	return json.Unmarshal(content, &s.data)
}

func (s *Store) save() error {
	// Internal save, lock held by caller
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return atomic.WriteFile(s.path, bytes.NewReader(b))
}

func (s *Store) GetAll() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]*Task, 0, len(s.data.Tasks))
	for _, t := range s.data.Tasks {
		// Return copy? For now pointer is okay as we don't mutate returned task directly
		// ideally deep copy
		tasks = append(tasks, t)
	}
	return tasks
}

func (s *Store) UpdateTask(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Tasks[t.ID] = t
	return s.save()
}

// LeaseTask attempts to acquire a lease for a task.
func (s *Store) LeaseTask(taskID, runID string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.data.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found")
	}

	// Check if already leased and valid
	if t.Lease != nil && t.Lease.Status == StatusLeased && time.Now().Before(t.Lease.ExpiresAt) {
		return fmt.Errorf("task already leased")
	}

	t.Lease = &Lease{
		RunID:     runID,
		Status:    StatusLeased,
		ExpiresAt: time.Now().Add(duration),
	}
	return s.save()
}

func (s *Store) CompleteTask(taskID string, nextRun time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.data.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found")
	}

	t.Lease = nil
	t.NextRun = nextRun
	return s.save()
}

func (s *Store) Init(ctx context.Context) error {
	return s.load()
}

func (s *Store) LoadTasks() ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.data.Tasks))
	for _, t := range s.data.Tasks {
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (s *Store) ShouldFire(taskID, schedule string) (bool, time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.data.Tasks[taskID]
	if !ok {
		return false, time.Time{}, fmt.Errorf("task not found")
	}

	if t.NextRun.After(time.Now()) {
		return false, t.NextRun, nil
	}

	cronSchedule, err := cron.ParseStandard(schedule)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("invalid cron schedule: %w", err)
	}

	nextRun := cronSchedule.Next(time.Now())
	t.NextRun = nextRun
	return true, nextRun, nil
}

func (s *Store) AcquireLease(taskID, runID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.data.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found")
	}

	if t.Lease != nil && t.Lease.Status == StatusLeased && time.Now().Before(t.Lease.ExpiresAt) {
		return fmt.Errorf("task already leased")
	}

	t.Lease = &Lease{
		RunID:     runID,
		Status:    StatusLeased,
		ExpiresAt: expiresAt,
	}
	return s.save()
}

func (s *Store) MarkTaskDone(taskID, runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.data.Tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found")
	}

	if t.Lease == nil || t.Lease.RunID != runID {
		return fmt.Errorf("lease mismatch")
	}

	t.Lease = nil
	cronSchedule, err := cron.ParseStandard(t.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}

	t.NextRun = cronSchedule.Next(time.Now())
	return s.save()
}

func (s *Store) GetLease(taskID string) (*Lease, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.data.Tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	return t.Lease, nil
}

func generateID() string {
	return ulid.Make().String()
}

func generateRunID() string {
	return ulid.Make().String()
}
