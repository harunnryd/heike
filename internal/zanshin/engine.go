package zanshin

import (
	"context"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
)

type Engine struct {
	cfg             config.ZanshinConfig
	maxIdle         time.Duration
	pollInterval    time.Duration
	queueSizer      func() int
	mu              sync.RWMutex
	started         bool
	lastInteraction time.Time
	lastRun         time.Time
	runCount        int
}

func NewEngine(cfg config.ZanshinConfig, queueSizer func() int) *Engine {
	maxIdle, err := config.DurationOrDefault(cfg.MaxIdleTime, config.DefaultZanshinMaxIdleTime)
	if err != nil {
		maxIdle, _ = config.DurationOrDefault("", config.DefaultZanshinMaxIdleTime)
	}
	if cfg.TriggerThreshold <= 0 {
		cfg.TriggerThreshold = config.DefaultZanshinTriggerThreshold
	}
	if cfg.PruneThreshold <= 0 {
		cfg.PruneThreshold = config.DefaultZanshinPruneThreshold
	}
	if cfg.SimilarityEpsilon <= 0 {
		cfg.SimilarityEpsilon = config.DefaultZanshinSimilarityEpsilon
	}
	if cfg.ClusterCount <= 0 {
		cfg.ClusterCount = config.DefaultZanshinClusterCount
	}

	return &Engine{
		cfg:             cfg,
		maxIdle:         maxIdle,
		pollInterval:    5 * time.Second,
		queueSizer:      queueSizer,
		lastInteraction: time.Now(),
	}
}

func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.started || !e.cfg.Enabled {
		e.mu.Unlock()
		return
	}
	e.started = true
	e.mu.Unlock()

	ticker := time.NewTicker(e.pollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				e.mu.Lock()
				e.started = false
				e.mu.Unlock()
				return
			case <-ticker.C:
				queueSize := 0
				if e.queueSizer != nil {
					queueSize = e.queueSizer()
				}
				e.mu.RLock()
				idle := time.Since(e.lastInteraction)
				e.mu.RUnlock()
				if e.ShouldTrigger(queueSize, 0, idle) {
					_ = e.Process(ctx)
				}
			}
		}
	}()
}

func (e *Engine) NotifyInteraction() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastInteraction = time.Now()
}

func (e *Engine) Process(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastRun = time.Now()
	e.runCount++
	return nil
}

func (e *Engine) ShouldTrigger(queueSize int, fatigue float64, idleTime time.Duration) bool {
	queueEmpty := 0.0
	if queueSize == 0 {
		queueEmpty = 1.0
	}
	idleFactor := idleTime.Minutes() / 30.0
	if e.maxIdle > 0 && idleTime >= e.maxIdle {
		idleFactor = 1
	}
	if idleFactor > 1 {
		idleFactor = 1
	}
	if fatigue < 0 {
		fatigue = 0
	}
	if fatigue > 1 {
		fatigue = 1
	}
	z := 0.4*queueEmpty + 0.3*fatigue + 0.3*idleFactor
	return z >= e.cfg.TriggerThreshold
}

func (e *Engine) Status() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return map[string]interface{}{
		"enabled":           e.cfg.Enabled,
		"started":           e.started,
		"trigger_threshold": e.cfg.TriggerThreshold,
		"prune_threshold":   e.cfg.PruneThreshold,
		"cluster_count":     e.cfg.ClusterCount,
		"last_run":          e.lastRun,
		"run_count":         e.runCount,
		"last_interaction":  e.lastInteraction,
	}
}
