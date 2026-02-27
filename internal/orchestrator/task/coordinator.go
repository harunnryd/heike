package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/config"
)

type Coordinator struct {
	engine       cognitive.Engine
	retryMax     int
	retryBackoff time.Duration
}

func NewCoordinator(engine cognitive.Engine, retryMax int, retryBackoff time.Duration) *Coordinator {
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

	return &Coordinator{
		engine:       engine,
		retryMax:     retryMax,
		retryBackoff: retryBackoff,
	}
}

type SubTaskResult struct {
	ID      string
	Success bool
	Output  string
	Error   error
}

// ExecuteDAG executes subtasks respecting their dependencies
func (c *Coordinator) ExecuteDAG(ctx context.Context, parentCtx *cognitive.CognitiveContext, subTasks []*SubTask) ([]SubTaskResult, error) {
	// Build Dependency Graph
	graph := make(map[string][]string)      // taskID -> []dependencyIDs
	dependents := make(map[string][]string) // taskID -> []dependentIDs
	results := make(map[string]SubTaskResult)
	var mu sync.Mutex

	taskMap := make(map[string]*SubTask)
	for _, t := range subTasks {
		taskMap[t.ID] = t
		graph[t.ID] = t.Dependencies
		for _, dep := range t.Dependencies {
			dependents[dep] = append(dependents[dep], t.ID)
		}
	}

	// Identify Initial Tasks (no dependencies)
	readyQueue := make(chan string, len(subTasks))
	remainingDeps := make(map[string]int)

	for id, deps := range graph {
		remainingDeps[id] = len(deps)
		if len(deps) == 0 {
			readyQueue <- id
		}
	}

	// Execution Loop
	var wg sync.WaitGroup

	// Track task completion
	completedTasks := 0
	totalTasks := len(subTasks)

	// Worker pool (limit concurrency if needed, for now unbounded within reason)
	go func() {
		for {
			mu.Lock()
			if completedTasks == totalTasks {
				mu.Unlock()
				close(readyQueue)
				return
			}

			mu.Unlock()

			select {
			case id, ok := <-readyQueue:
				if !ok {
					return
				}

				wg.Add(1)
				go func(tid string) {
					defer wg.Done()

					t := taskMap[tid]

					subCtxOpts := func(c *cognitive.CognitiveContext) {
						c.SessionID = parentCtx.SessionID
						c.WorkspaceID = parentCtx.WorkspaceID
						c.AvailableTools = parentCtx.AvailableTools
						c.AvailableSkills = append([]string(nil), parentCtx.AvailableSkills...)
						if len(parentCtx.Metadata) > 0 {
							if c.Metadata == nil {
								c.Metadata = make(map[string]string, len(parentCtx.Metadata))
							}
							for key, value := range parentCtx.Metadata {
								c.Metadata[key] = value
							}
						}

						// Add context from dependencies
						// This is a "Senior" feature: context chaining
						for _, depID := range t.Dependencies {
							if res, ok := results[depID]; ok && res.Success {
								// Naive injection, in reality we might want structured IO
								slog.Debug("Injecting dependency context", "task", tid, "dep", depID)
								c.Scratchpad = append(c.Scratchpad, fmt.Sprintf("Dependency %s Output: %s", depID, res.Output))
							}
						}
					}

					start := time.Now()
					slog.Info("Starting sub-task", "id", t.ID, "desc", t.Description)

					// Retry Logic (Simple)
					var res *cognitive.Result
					var err error
					for attempt := 0; attempt < c.retryMax; attempt++ {
						res, err = c.engine.Run(ctx, t.Description, subCtxOpts)
						if err == nil {
							break
						}
						slog.Warn("Sub-task retry", "id", t.ID, "attempt", attempt+1, "error", err)
						time.Sleep(c.retryBackoff * time.Duration(attempt+1))
					}

					_ = time.Since(start)

					mu.Lock()
					defer mu.Unlock()

					if err != nil {
						slog.Error("Sub-task failed", "id", t.ID, "error", err)
						results[tid] = SubTaskResult{ID: tid, Success: false, Error: err}
					} else {
						slog.Info("Sub-task completed", "id", t.ID)
						results[tid] = SubTaskResult{ID: tid, Success: true, Output: res.Content}

						// Unlock dependents
						for _, depID := range dependents[tid] {
							remainingDeps[depID]--
							if remainingDeps[depID] == 0 {
								readyQueue <- depID
							}
						}
					}
					completedTasks++
				}(id)

			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	// Convert map to slice
	finalResults := make([]SubTaskResult, 0, len(results))
	for _, r := range results {
		finalResults = append(finalResults, r)
	}

	return finalResults, nil
}
