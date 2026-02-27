package task

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/config"
)

type Coordinator struct {
	engine       cognitive.Engine
	retryMax     int
	retryBackoff time.Duration
	maxParallel  int
}

func NewCoordinator(engine cognitive.Engine, retryMax int, retryBackoff time.Duration, maxParallel int) *Coordinator {
	if retryMax <= 0 {
		retryMax = config.DefaultOrchestratorSubTaskRetryMax
	}
	if retryBackoff <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultOrchestratorSubTaskRetryBackoff)
		if err == nil {
			retryBackoff = d
		}
	}
	if retryBackoff <= 0 {
		retryBackoff = time.Second
	}
	if maxParallel <= 0 {
		maxParallel = config.DefaultOrchestratorMaxParallelSubTasks
	}
	if maxParallel <= 0 {
		maxParallel = 1
	}

	return &Coordinator{
		engine:       engine,
		retryMax:     retryMax,
		retryBackoff: retryBackoff,
		maxParallel:  maxParallel,
	}
}

type SubTaskResult struct {
	ID      string
	Success bool
	Output  string
	Error   error
}

// ExecuteDAG executes subtasks in deterministic topological batches.
func (c *Coordinator) ExecuteDAG(ctx context.Context, parentCtx *cognitive.CognitiveContext, subTasks []*SubTask) ([]SubTaskResult, error) {
	if len(subTasks) == 0 {
		return nil, nil
	}

	batches, err := resolveExecutionBatches(subTasks)
	if err != nil {
		return nil, err
	}

	resultsByID := make(map[string]SubTaskResult, len(subTasks))
	ordered := make([]SubTaskResult, 0, len(subTasks))

	for _, batch := range batches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		batchResults, err := c.executeBatch(ctx, parentCtx, batch, resultsByID)
		if err != nil {
			return nil, err
		}
		for _, res := range batchResults {
			resultsByID[res.ID] = res
			ordered = append(ordered, res)
		}
	}

	return ordered, nil
}

func (c *Coordinator) executeBatch(
	ctx context.Context,
	parentCtx *cognitive.CognitiveContext,
	batch []*SubTask,
	resultsByID map[string]SubTaskResult,
) ([]SubTaskResult, error) {
	sem := make(chan struct{}, c.maxParallel)
	batchResultByID := make(map[string]SubTaskResult, len(batch))

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, task := range batch {
		t := task
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				mu.Lock()
				batchResultByID[t.ID] = SubTaskResult{ID: t.ID, Success: false, Error: ctx.Err()}
				mu.Unlock()
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			res := c.executeTask(ctx, parentCtx, t, resultsByID)

			mu.Lock()
			batchResultByID[t.ID] = res
			mu.Unlock()
		}()
	}

	wg.Wait()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ordered := make([]SubTaskResult, 0, len(batch))
	for _, task := range batch {
		ordered = append(ordered, batchResultByID[task.ID])
	}
	return ordered, nil
}

func (c *Coordinator) executeTask(
	ctx context.Context,
	parentCtx *cognitive.CognitiveContext,
	t *SubTask,
	resultsByID map[string]SubTaskResult,
) SubTaskResult {
	for _, depID := range t.Dependencies {
		depRes, ok := resultsByID[depID]
		if !ok {
			return SubTaskResult{ID: t.ID, Success: false, Error: fmt.Errorf("dependency %s missing result", depID)}
		}
		if !depRes.Success {
			return SubTaskResult{ID: t.ID, Success: false, Error: fmt.Errorf("dependency %s failed", depID)}
		}
	}

	subCtxOpts := func(cCtx *cognitive.CognitiveContext) {
		cCtx.SessionID = parentCtx.SessionID
		cCtx.WorkspaceID = parentCtx.WorkspaceID
		cCtx.AvailableTools = parentCtx.AvailableTools
		cCtx.AvailableSkills = append([]string(nil), parentCtx.AvailableSkills...)
		if len(parentCtx.Metadata) > 0 {
			if cCtx.Metadata == nil {
				cCtx.Metadata = make(map[string]string, len(parentCtx.Metadata))
			}
			for key, value := range parentCtx.Metadata {
				cCtx.Metadata[key] = value
			}
		}

		for _, depID := range t.Dependencies {
			if depRes, ok := resultsByID[depID]; ok && depRes.Success {
				cCtx.Scratchpad = append(cCtx.Scratchpad, fmt.Sprintf("Dependency %s Output: %s", depID, depRes.Output))
			}
		}
	}

	slog.Info("Starting sub-task", "id", t.ID, "desc", t.Description)

	var lastErr error
	for attempt := 0; attempt < c.retryMax; attempt++ {
		select {
		case <-ctx.Done():
			return SubTaskResult{ID: t.ID, Success: false, Error: ctx.Err()}
		default:
		}

		res, err := c.engine.Run(ctx, t.Description, subCtxOpts)
		if err == nil {
			slog.Info("Sub-task completed", "id", t.ID)
			return SubTaskResult{ID: t.ID, Success: true, Output: res.Content}
		}

		lastErr = err
		slog.Warn("Sub-task retry", "id", t.ID, "attempt", attempt+1, "error", err)

		if attempt < c.retryMax-1 {
			backoff := c.retryBackoff * time.Duration(attempt+1)
			select {
			case <-ctx.Done():
				return SubTaskResult{ID: t.ID, Success: false, Error: ctx.Err()}
			case <-time.After(backoff):
			}
		}
	}

	slog.Error("Sub-task failed", "id", t.ID, "error", lastErr)
	return SubTaskResult{ID: t.ID, Success: false, Error: lastErr}
}

func resolveExecutionBatches(subTasks []*SubTask) ([][]*SubTask, error) {
	taskByID := make(map[string]*SubTask, len(subTasks))
	inDegree := make(map[string]int, len(subTasks))
	dependents := make(map[string][]string, len(subTasks))

	for _, task := range subTasks {
		if task == nil {
			return nil, fmt.Errorf("nil sub-task")
		}
		if task.ID == "" {
			return nil, fmt.Errorf("sub-task with empty id")
		}
		if _, exists := taskByID[task.ID]; exists {
			return nil, fmt.Errorf("duplicate sub-task id: %s", task.ID)
		}
		taskByID[task.ID] = task
		inDegree[task.ID] = 0
	}

	for _, task := range subTasks {
		for _, depID := range task.Dependencies {
			if _, exists := taskByID[depID]; !exists {
				return nil, fmt.Errorf("sub-task %s depends on unknown task %s", task.ID, depID)
			}
			if depID == task.ID {
				return nil, fmt.Errorf("sub-task %s cannot depend on itself", task.ID)
			}
			inDegree[task.ID]++
			dependents[depID] = append(dependents[depID], task.ID)
		}
	}

	ready := make([]string, 0, len(subTasks))
	for id, deg := range inDegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)

	processed := 0
	batches := make([][]*SubTask, 0)

	for len(ready) > 0 {
		currentIDs := append([]string(nil), ready...)
		ready = ready[:0]

		batch := make([]*SubTask, 0, len(currentIDs))
		for _, id := range currentIDs {
			batch = append(batch, taskByID[id])
		}
		batches = append(batches, batch)
		processed += len(currentIDs)

		nextSet := make(map[string]struct{})
		for _, id := range currentIDs {
			for _, depID := range dependents[id] {
				inDegree[depID]--
				if inDegree[depID] == 0 {
					nextSet[depID] = struct{}{}
				}
			}
		}

		for id := range nextSet {
			ready = append(ready, id)
		}
		sort.Strings(ready)
	}

	if processed != len(subTasks) {
		return nil, fmt.Errorf("circular dependency detected in sub-tasks")
	}

	return batches, nil
}
