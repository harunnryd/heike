package builtin

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultExecYieldMS    = 100
	maxSessionBufferBytes = 1 << 20 // 1 MiB
)

type execSessionStore struct {
	mu       sync.RWMutex
	nextID   int64
	sessions map[int64]*execSession
}

type execSession struct {
	id int64

	cmd   *exec.Cmd
	stdin io.WriteCloser

	mu      sync.Mutex
	output  []byte
	readPos int

	done     chan struct{}
	exitCode int
}

var globalExecSessions = &execSessionStore{
	nextID:   0,
	sessions: make(map[int64]*execSession),
}

func startExecSession(cmd *exec.Cmd) (int64, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return 0, err
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	id := atomic.AddInt64(&globalExecSessions.nextID, 1)
	s := &execSession{
		id:       id,
		cmd:      cmd,
		stdin:    stdin,
		done:     make(chan struct{}),
		exitCode: 0,
	}

	globalExecSessions.mu.Lock()
	globalExecSessions.sessions[id] = s
	globalExecSessions.mu.Unlock()

	go s.capture(stdout)
	go s.capture(stderr)
	go s.waitAndClose()

	return id, nil
}

func getExecSession(id int64) (*execSession, bool) {
	globalExecSessions.mu.RLock()
	defer globalExecSessions.mu.RUnlock()
	s, ok := globalExecSessions.sessions[id]
	return s, ok
}

func deleteExecSession(id int64) {
	globalExecSessions.mu.Lock()
	defer globalExecSessions.mu.Unlock()
	delete(globalExecSessions.sessions, id)
}

func (s *execSession) capture(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s.appendOutput(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (s *execSession) appendOutput(chunk []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.output = append(s.output, chunk...)
	if len(s.output) <= maxSessionBufferBytes {
		return
	}

	drop := len(s.output) - maxSessionBufferBytes
	s.output = s.output[drop:]
	if s.readPos > drop {
		s.readPos -= drop
	} else {
		s.readPos = 0
	}
}

func (s *execSession) waitAndClose() {
	err := s.cmd.Wait()

	s.mu.Lock()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			s.exitCode = exitErr.ExitCode()
		} else {
			s.exitCode = -1
			s.output = append(s.output, []byte(fmt.Sprintf("\nprocess error: %v\n", err))...)
		}
	}
	s.mu.Unlock()

	close(s.done)

	// Auto-cleanup stale completed sessions to avoid memory growth.
	time.AfterFunc(2*time.Minute, func() {
		deleteExecSession(s.id)
	})
}

func (s *execSession) write(chars string) error {
	select {
	case <-s.done:
		return fmt.Errorf("session is closed")
	default:
	}

	if chars == "" {
		return nil
	}
	_, err := io.WriteString(s.stdin, chars)
	return err
}

func (s *execSession) readNewOutput(maxOutputTokens int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.readPos >= len(s.output) {
		return ""
	}
	newChunk := string(s.output[s.readPos:])
	s.readPos = len(s.output)
	return truncateOutputByTokens(newChunk, maxOutputTokens)
}

func (s *execSession) running() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

func (s *execSession) getExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}

func normalizeYieldDuration(ms int) time.Duration {
	if ms <= 0 {
		ms = defaultExecYieldMS
	}
	return time.Duration(ms) * time.Millisecond
}

func truncateOutputByTokens(output string, maxOutputTokens int) string {
	if maxOutputTokens <= 0 {
		return output
	}
	approxChars := maxOutputTokens * 4
	if approxChars <= 0 || len(output) <= approxChars {
		return output
	}
	return output[len(output)-approxChars:]
}
