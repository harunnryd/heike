package zanshin

import (
	"context"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/config"
)

func TestEngine_ShouldTrigger(t *testing.T) {
	engine := NewEngine(config.ZanshinConfig{
		Enabled:          true,
		TriggerThreshold: 0.5,
		MaxIdleTime:      "30m",
	}, nil)

	if !engine.ShouldTrigger(0, 0.2, 20*time.Minute) {
		t.Fatal("expected trigger when queue empty and system is idle")
	}
	if engine.ShouldTrigger(10, 0.0, 0) {
		t.Fatal("did not expect trigger when queue is busy and no fatigue")
	}
}

func TestEngine_StartAndStatus(t *testing.T) {
	engine := NewEngine(config.ZanshinConfig{
		Enabled:          true,
		TriggerThreshold: 0.1,
		MaxIdleTime:      "1m",
	}, func() int { return 0 })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	engine.NotifyInteraction()
	_ = engine.Process(ctx)

	status := engine.Status()
	if status["enabled"] != true {
		t.Fatalf("expected enabled true, got %v", status["enabled"])
	}
	if status["run_count"].(int) < 1 {
		t.Fatalf("expected run_count >= 1, got %v", status["run_count"])
	}
}
