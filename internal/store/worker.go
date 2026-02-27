package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	stdatomic "sync/atomic"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/idempotency"

	"github.com/natefinch/atomic"
	"github.com/philippgille/chromem-go"
)

type Operation int

const (
	OpWriteTranscript Operation = iota
	OpSaveIdempotency
	OpResetSession
	OpGetSession
	OpSaveSession
	OpUpsertVector
	OpSearchVectors
	OpReadTranscript
)

type Request struct {
	Op       Operation
	Payload  interface{}
	Result   chan error
	Response chan interface{}
}

type TranscriptPayload struct {
	SessionID string
	Data      []byte // JSON line
}

type ResetSessionPayload struct {
	SessionID string
}

type GetSessionPayload struct {
	SessionID string
}

type SaveSessionPayload struct {
	Session *SessionMeta
}

type UpsertVectorPayload struct {
	Collection string
	ID         string
	Vector     []float32
	Metadata   map[string]string
	Content    string
}

type SearchVectorsPayload struct {
	Collection string
	Vector     []float32
	Limit      int
}

type ReadTranscriptPayload struct {
	SessionID string
	Limit     int // 0 = all
}

type VectorResult struct {
	ID       string
	Score    float32
	Metadata map[string]string
	Content  string
}

type Worker struct {
	workspaceID              string
	basePath                 string
	inbox                    chan Request
	idemStore                *idempotency.Store
	fileLock                 *FileLock
	quit                     chan struct{}
	wg                       sync.WaitGroup
	sessionIndex             *SessionIndex
	vectorDB                 *chromem.DB
	running                  stdatomic.Bool
	transcriptRotateMaxBytes int64
}

type RuntimeConfig struct {
	LockTimeout              time.Duration
	LockRetry                time.Duration
	LockMaxRetry             int
	InboxSize                int
	TranscriptRotateMaxBytes int64
}

func NewWorker(workspaceID string, workspaceRootPath string, runtimeCfg RuntimeConfig) (*Worker, error) {
	basePath, err := GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return nil, err
	}

	// Init Directories
	dirs := []string{
		filepath.Join(basePath, "sessions"),
		filepath.Join(basePath, "governance"),
		filepath.Join(basePath, "sandbox"),
		filepath.Join(basePath, "scheduler"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create dir %s: %w", d, err)
		}
	}

	if runtimeCfg.LockTimeout <= 0 {
		lockTimeout, err := config.DurationOrDefault("", config.DefaultStoreLockTimeout)
		if err != nil {
			return nil, fmt.Errorf("parse default store lock timeout: %w", err)
		}
		runtimeCfg.LockTimeout = lockTimeout
	}
	if runtimeCfg.LockRetry <= 0 {
		lockRetry, err := config.DurationOrDefault("", config.DefaultStoreLockRetry)
		if err != nil {
			return nil, fmt.Errorf("parse default store lock retry: %w", err)
		}
		runtimeCfg.LockRetry = lockRetry
	}
	if runtimeCfg.LockMaxRetry <= 0 {
		runtimeCfg.LockMaxRetry = config.DefaultStoreLockMaxRetry
	}
	if runtimeCfg.InboxSize <= 0 {
		runtimeCfg.InboxSize = config.DefaultStoreInboxSize
	}
	if runtimeCfg.TranscriptRotateMaxBytes <= 0 {
		runtimeCfg.TranscriptRotateMaxBytes = config.DefaultStoreTranscriptRotateMaxBytes
	}

	// File Lock (Single Instance per Workspace)
	fileLock, err := NewFileLock(workspaceID, basePath, &FileLockConfig{
		LockTimeout:  runtimeCfg.LockTimeout,
		LockRetry:    runtimeCfg.LockRetry,
		LockMaxRetry: runtimeCfg.LockMaxRetry,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Load Idempotency Store
	idemPath := filepath.Join(basePath, "governance", "processed_keys.json")
	idemStore, err := idempotency.NewStore(idemPath)
	if err != nil {
		fileLock.Unlock()
		return nil, fmt.Errorf("failed to load idempotency store: %w", err)
	}

	// Load Session Index
	sessionIndex := &SessionIndex{Sessions: make(map[string]SessionMeta)}
	indexPath := filepath.Join(basePath, "sessions", "index.json")
	if data, err := os.ReadFile(indexPath); err == nil {
		if err := json.Unmarshal(data, sessionIndex); err != nil {
			slog.Warn("Failed to parse session index, starting fresh", "error", err)
		}
	}

	// Init Vector DB
	vectorPath := filepath.Join(basePath, "vectors")
	if err := os.MkdirAll(vectorPath, 0755); err != nil {
		fileLock.Unlock()
		return nil, fmt.Errorf("failed to create vector dir: %w", err)
	}
	// Using empty string for embedding func because we provide embeddings manually
	vectorDB, err := chromem.NewPersistentDB(vectorPath, false)
	if err != nil {
		fileLock.Unlock()
		return nil, fmt.Errorf("failed to init vector db: %w", err)
	}

	return &Worker{
		workspaceID:              workspaceID,
		basePath:                 basePath,
		inbox:                    make(chan Request, runtimeCfg.InboxSize),
		idemStore:                idemStore,
		fileLock:                 fileLock,
		quit:                     make(chan struct{}),
		sessionIndex:             sessionIndex,
		vectorDB:                 vectorDB,
		transcriptRotateMaxBytes: runtimeCfg.TranscriptRotateMaxBytes,
	}, nil
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go w.loop()
}

func (w *Worker) loop() {
	slog.Info("StoreWorker started", "workspace", w.workspaceID)
	w.running.Store(true)
	defer func() {
		w.running.Store(false)
		w.wg.Done()
	}()

	// Initial Prune
	pruned := w.idemStore.Prune()
	if pruned > 0 {
		slog.Info("Pruned expired idempotency keys", "count", pruned)
		if err := w.idemStore.Save(); err != nil {
			slog.Error("Failed to save pruned keys", "error", err)
		}
	}

	for {
		select {
		case req := <-w.inbox:
			err := w.handle(req)
			if req.Result != nil {
				req.Result <- err
			}
		case <-w.quit:
			slog.Info("StoreWorker stopping")
			return
		}
	}
}

func (w *Worker) handle(req Request) error {
	switch req.Op {
	case OpWriteTranscript:
		p, ok := req.Payload.(TranscriptPayload)
		if !ok {
			return fmt.Errorf("invalid payload for WriteTranscript")
		}
		return w.appendTranscript(p.SessionID, p.Data)
	case OpSaveIdempotency:
		return w.idemStore.Save()
	case OpResetSession:
		p, ok := req.Payload.(ResetSessionPayload)
		if !ok {
			return fmt.Errorf("invalid payload for ResetSession")
		}
		return w.resetSession(p.SessionID)
	case OpGetSession:
		p, ok := req.Payload.(GetSessionPayload)
		if !ok {
			return fmt.Errorf("invalid payload for GetSession")
		}
		if sess, ok := w.sessionIndex.Sessions[p.SessionID]; ok {
			if req.Response != nil {
				req.Response <- &sess
			}
		} else {
			if req.Response != nil {
				req.Response <- nil
			}
		}
		return nil
	case OpSaveSession:
		p, ok := req.Payload.(SaveSessionPayload)
		if !ok {
			return fmt.Errorf("invalid payload for SaveSession")
		}
		w.sessionIndex.Sessions[p.Session.ID] = *p.Session
		return w.saveSessionIndex()
	case OpUpsertVector:
		p, ok := req.Payload.(UpsertVectorPayload)
		if !ok {
			return fmt.Errorf("invalid payload for UpsertVector")
		}
		return w.upsertVector(p)
	case OpSearchVectors:
		p, ok := req.Payload.(SearchVectorsPayload)
		if !ok {
			return fmt.Errorf("invalid payload for SearchVectors")
		}
		res, err := w.searchVectors(p)
		if req.Response != nil {
			req.Response <- res
		}
		return err
	case OpReadTranscript:
		p, ok := req.Payload.(ReadTranscriptPayload)
		if !ok {
			return fmt.Errorf("invalid payload for ReadTranscript")
		}
		lines, err := w.readTranscript(p.SessionID, p.Limit)
		if req.Response != nil {
			req.Response <- lines
		}
		return err
	default:
		return fmt.Errorf("unknown operation: %d", req.Op)
	}
}

func (w *Worker) readTranscript(sessionID string, limit int) ([]string, error) {
	path := filepath.Join(w.basePath, "sessions", sessionID+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}

	if limit > 0 && len(lines) > limit {
		// Return last N lines
		return lines[len(lines)-limit:], nil
	}
	return lines, nil
}

func (w *Worker) upsertVector(p UpsertVectorPayload) error {
	// Nil embedding func because we provide embeddings
	col, err := w.vectorDB.GetOrCreateCollection(p.Collection, nil, nil)
	if err != nil {
		return err
	}
	// AddDocuments is upsert in chromem
	return col.AddDocuments(context.Background(), []chromem.Document{
		{
			ID:        p.ID,
			Metadata:  p.Metadata,
			Embedding: p.Vector,
			Content:   p.Content,
		},
	}, 1) // parallelism = 1 for safety in store worker
}

func (w *Worker) searchVectors(p SearchVectorsPayload) ([]VectorResult, error) {
	col := w.vectorDB.GetCollection(p.Collection, nil)
	if col == nil {
		// Collection doesn't exist yet, return empty
		return []VectorResult{}, nil
	}

	// QueryEmbedding(ctx, embedding, nResults, where, whereDocument)
	docs, err := col.QueryEmbedding(context.Background(), p.Vector, p.Limit, nil, nil)
	if err != nil {
		return nil, err
	}

	var results []VectorResult
	for _, doc := range docs {
		results = append(results, VectorResult{
			ID:       doc.ID,
			Score:    doc.Similarity,
			Metadata: doc.Metadata,
			Content:  doc.Content,
		})
	}
	return results, nil
}

func (w *Worker) saveSessionIndex() error {
	path := filepath.Join(w.basePath, "sessions", "index.json")
	data, err := json.MarshalIndent(w.sessionIndex, "", "  ")
	if err != nil {
		return err
	}
	return atomic.WriteFile(path, bytes.NewReader(data))
}

func (w *Worker) appendTranscript(sessionID string, data []byte) error {
	path := filepath.Join(w.basePath, "sessions", sessionID+".jsonl")

	if err := w.checkAndRotate(sessionID, path); err != nil {
		slog.Warn("Failed to rotate transcript", "session", sessionID, "error", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}
	return f.Sync()
}

func (w *Worker) resetSession(sessionID string) error {
	path := filepath.Join(w.basePath, "sessions", sessionID+".jsonl")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Remove from index
	delete(w.sessionIndex.Sessions, sessionID)
	return w.saveSessionIndex()
}

func (w *Worker) checkAndRotate(sessionID, path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if info.Size() < w.transcriptRotateMaxBytes {
		return nil
	}

	slog.Info("Rotating transcript", "session", sessionID, "size", info.Size())

	// Rotate
	timestamp := time.Now().Format("20060102150405")
	backupPath := fmt.Sprintf("%s.%s.bak", path, timestamp)

	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	// Create new empty file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create new transcript: %w", err)
	}
	f.Close()

	return nil
}

// Public API for other components

func (w *Worker) WriteTranscript(sessionID string, data []byte) error {
	res := make(chan error, 1)
	w.inbox <- Request{
		Op:      OpWriteTranscript,
		Payload: TranscriptPayload{SessionID: sessionID, Data: data},
		Result:  res,
	}
	return <-res
}

func (w *Worker) ResetSession(sessionID string) error {
	res := make(chan error, 1)
	w.inbox <- Request{
		Op:      OpResetSession,
		Payload: ResetSessionPayload{SessionID: sessionID},
		Result:  res,
	}
	return <-res
}

// ListSessions lists all session IDs in the workspace.
// This is a direct read operation, safe if concurrent with writes as file system handles dir listing.
func (w *Worker) ListSessions() ([]string, error) {
	// Better to use the index now
	// But let's stick to filesystem scan for robustness or use index?
	// The docs say index.json is the registry.
	// But let's keep it consistent with what we have in memory.
	// Since GetSession uses memory, ListSessions should too.
	// However, ListSessions is synchronous and runs in the caller's goroutine.
	// Accessing w.sessionIndex (map) concurrently is unsafe!
	// So ListSessions MUST go through the worker loop or use a mutex.
	// The previous implementation used os.ReadDir which is safe.
	// Let's stick to os.ReadDir for now to avoid breaking changes, or implement OpListSessions.
	// Given we have the index in memory, let's use it but safely.
	// Actually, let's keep the os.ReadDir implementation as it is robust against manual file deletions.
	// But wait, if I use index.json, I should rely on it.
	// For now, let's keep os.ReadDir as it was "safe" before.
	// But strict adherence to architecture says "Use Index".
	// Let's implement OpListSessions for correctness.

	// For now, let's keep the old implementation to minimize friction, but note that it might be out of sync with index.
	sessionsDir := filepath.Join(w.basePath, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			id := strings.TrimSuffix(entry.Name(), ".jsonl")
			sessions = append(sessions, id)
		}
	}
	return sessions, nil
}

func (w *Worker) GetSession(id string) (*SessionMeta, error) {
	res := make(chan error, 1)
	resp := make(chan interface{}, 1)
	w.inbox <- Request{
		Op:       OpGetSession,
		Payload:  GetSessionPayload{SessionID: id},
		Result:   res,
		Response: resp,
	}
	if err := <-res; err != nil {
		return nil, err
	}
	val := <-resp
	if val == nil {
		return nil, nil // Not found
	}
	return val.(*SessionMeta), nil
}

func (w *Worker) SaveSession(session *SessionMeta) error {
	res := make(chan error, 1)
	w.inbox <- Request{
		Op:      OpSaveSession,
		Payload: SaveSessionPayload{Session: session},
		Result:  res,
	}
	return <-res
}

func (w *Worker) UpsertVector(collection, id string, vector []float32, metadata map[string]string, content string) error {
	res := make(chan error, 1)
	w.inbox <- Request{
		Op: OpUpsertVector,
		Payload: UpsertVectorPayload{
			Collection: collection,
			ID:         id,
			Vector:     vector,
			Metadata:   metadata,
			Content:    content,
		},
		Result: res,
	}
	return <-res
}

func (w *Worker) SearchVectors(collection string, vector []float32, limit int) ([]VectorResult, error) {
	res := make(chan error, 1)
	resp := make(chan interface{}, 1)
	w.inbox <- Request{
		Op: OpSearchVectors,
		Payload: SearchVectorsPayload{
			Collection: collection,
			Vector:     vector,
			Limit:      limit,
		},
		Result:   res,
		Response: resp,
	}
	if err := <-res; err != nil {
		return nil, err
	}
	val := <-resp
	return val.([]VectorResult), nil
}

func (w *Worker) ReadTranscript(sessionID string, limit int) ([]string, error) {
	res := make(chan error, 1)
	resp := make(chan interface{}, 1)
	w.inbox <- Request{
		Op: OpReadTranscript,
		Payload: ReadTranscriptPayload{
			SessionID: sessionID,
			Limit:     limit,
		},
		Result:   res,
		Response: resp,
	}
	if err := <-res; err != nil {
		return nil, err
	}
	val := <-resp
	return val.([]string), nil
}

func (w *Worker) SaveIdempotency() {
	// Fire and forget usually, but we might want to block if critical
	w.inbox <- Request{
		Op:     OpSaveIdempotency,
		Result: nil,
	}
}

func (w *Worker) SaveIdempotencySync() error {
	// Blocking version for tests or critical operations
	res := make(chan error, 1)
	w.inbox <- Request{
		Op:     OpSaveIdempotency,
		Result: res,
	}
	return <-res
}

func (w *Worker) CheckAndMarkKey(key string, ttl time.Duration) bool {
	// This is safe to call concurrently because idemStore uses a mutex
	// However, persistence is async via SaveIdempotency
	if ttl <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultGovernanceIdempotencyTTL)
		if err == nil {
			ttl = d
		}
	}
	exists := w.idemStore.CheckAndMark(key, ttl)
	if !exists {
		// Queue a save
		w.SaveIdempotency()
	}
	return exists
}

func (w *Worker) Stop() {
	slog.Info("StoreWorker Stop called", "workspace", w.workspaceID, "lock_held", w.fileLock.IsLocked())

	close(w.quit)
	w.wg.Wait()

	if w.fileLock.IsLocked() {
		w.fileLock.Unlock()
	}
}

func (w *Worker) IsLockHeld() bool {
	return w.fileLock.IsLocked()
}

func (w *Worker) IsRunning() bool {
	return w.fileLock.IsLocked() && w.running.Load()
}
