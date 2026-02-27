package task

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/cognitive"
)

type coordinatorTestEngine struct {
	runFn func(ctx context.Context, goal string) (*cognitive.Result, error)
}

func (e *coordinatorTestEngine) Run(ctx context.Context, goal string, opts ...cognitive.ExecutionOption) (*cognitive.Result, error) {
	if e.runFn != nil {
		return e.runFn(ctx, goal)
	}
	return &cognitive.Result{Content: goal}, nil
}

func TestResolveExecutionBatches_DeterministicOrder(t *testing.T) {
	subTasks := []*SubTask{
		{ID: "c", Description: "third", Dependencies: []string{"a", "b"}},
		{ID: "b", Description: "second"},
		{ID: "a", Description: "first"},
		{ID: "d", Description: "fourth", Dependencies: []string{"c"}},
	}

	batches, err := resolveExecutionBatches(subTasks)
	if err != nil {
		t.Fatalf("resolve execution batches: %v", err)
	}

	if len(batches) != 3 {
		t.Fatalf("batch count = %d, want 3", len(batches))
	}
	if batches[0][0].ID != "a" || batches[0][1].ID != "b" {
		t.Fatalf("batch[0] order = [%s,%s], want [a,b]", batches[0][0].ID, batches[0][1].ID)
	}
	if batches[1][0].ID != "c" {
		t.Fatalf("batch[1][0] = %s, want c", batches[1][0].ID)
	}
	if batches[2][0].ID != "d" {
		t.Fatalf("batch[2][0] = %s, want d", batches[2][0].ID)
	}
}

func TestCoordinator_ExecuteDAG_BoundedConcurrency(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32

	engine := &coordinatorTestEngine{
		runFn: func(ctx context.Context, goal string) (*cognitive.Result, error) {
			current := active.Add(1)
			for {
				prev := maxActive.Load()
				if current <= prev || maxActive.CompareAndSwap(prev, current) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			active.Add(-1)
			return &cognitive.Result{Content: fmt.Sprintf("%s done", goal)}, nil
		},
	}
	coord := NewCoordinator(engine, 1, 10*time.Millisecond, 2)

	subTasks := []*SubTask{
		{ID: "a", Description: "task-a"},
		{ID: "b", Description: "task-b"},
		{ID: "c", Description: "task-c"},
		{ID: "d", Description: "task-d"},
		{ID: "e", Description: "task-e"},
		{ID: "f", Description: "task-f"},
	}
	parentCtx := &cognitive.CognitiveContext{
		SessionID:   "session-1",
		WorkspaceID: "workspace-1",
	}

	results, err := coord.ExecuteDAG(context.Background(), parentCtx, subTasks)
	if err != nil {
		t.Fatalf("execute dag: %v", err)
	}
	if len(results) != len(subTasks) {
		t.Fatalf("result count = %d, want %d", len(results), len(subTasks))
	}
	for _, res := range results {
		if !res.Success {
			t.Fatalf("subtask %s failed unexpectedly: %v", res.ID, res.Error)
		}
	}
	if got := maxActive.Load(); got > 2 {
		t.Fatalf("max active subtasks = %d, want <= 2", got)
	}
}

func TestCoordinator_ExecuteDAG_DependencyFailurePropagation(t *testing.T) {
	engine := &coordinatorTestEngine{
		runFn: func(ctx context.Context, goal string) (*cognitive.Result, error) {
			if goal == "task-a" {
				return nil, fmt.Errorf("engine failure")
			}
			return &cognitive.Result{Content: goal + " done"}, nil
		},
	}
	coord := NewCoordinator(engine, 1, time.Millisecond, 2)

	subTasks := []*SubTask{
		{ID: "a", Description: "task-a"},
		{ID: "b", Description: "task-b", Dependencies: []string{"a"}},
	}
	parentCtx := &cognitive.CognitiveContext{
		SessionID:   "session-2",
		WorkspaceID: "workspace-1",
	}

	results, err := coord.ExecuteDAG(context.Background(), parentCtx, subTasks)
	if err != nil {
		t.Fatalf("execute dag: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if results[0].Success {
		t.Fatalf("expected task a to fail")
	}
	if results[1].Success {
		t.Fatalf("expected task b to fail due to dependency")
	}
	if results[1].Error == nil || !strings.Contains(results[1].Error.Error(), "dependency a failed") {
		t.Fatalf("unexpected dependency error: %v", results[1].Error)
	}
}
