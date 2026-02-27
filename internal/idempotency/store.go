package idempotency

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/natefinch/atomic"
)

type ProcessedKeys struct {
	Keys map[string]int64 `json:"keys"` // Key -> Expiry (Unix Timestamp)
}

type Store struct {
	path  string
	state ProcessedKeys
	mu    sync.RWMutex
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path: path,
		state: ProcessedKeys{
			Keys: make(map[string]int64),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s.save()
	}

	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &s.state)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	return atomic.WriteFile(s.path, bytes.NewReader(data))
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

func (s *Store) CheckAndMark(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()

	if expiry, exists := s.state.Keys[key]; exists {
		if expiry > now {
			return true
		}
		delete(s.state.Keys, key)
	}

	s.state.Keys[key] = now + int64(ttl.Seconds())
	return false
}

func (s *Store) Prune() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	count := 0
	for k, expiry := range s.state.Keys {
		if expiry < now {
			delete(s.state.Keys, k)
			count++
		}
	}
	return count
}
