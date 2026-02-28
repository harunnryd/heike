package components

import (
	"testing"

	"github.com/harunnryd/heike/internal/config"
)

func TestNewHTTPServerComponent_DefaultDependencies(t *testing.T) {
	comp := NewHTTPServerComponent(nil, &config.ServerConfig{Port: 8080})
	deps := comp.Dependencies()

	want := []string{"Runtime"}
	if len(deps) != len(want) {
		t.Fatalf("dependencies length = %d, want %d", len(deps), len(want))
	}
	for i := range want {
		if deps[i] != want[i] {
			t.Fatalf("dependency[%d] = %s, want %s", i, deps[i], want[i])
		}
	}
}

func TestNewHTTPServerComponentWithDependencies_Copy(t *testing.T) {
	custom := []string{"Runtime"}
	comp := NewHTTPServerComponentWithDependencies(nil, &config.ServerConfig{Port: 8080}, custom)

	custom[0] = "Mutated"

	deps := comp.Dependencies()
	if len(deps) != 1 {
		t.Fatalf("dependencies length = %d, want 1", len(deps))
	}
	if deps[0] != "Runtime" {
		t.Fatalf("dependency = %s, want Runtime", deps[0])
	}

	deps[0] = "MutatedAgain"
	if comp.Dependencies()[0] != "Runtime" {
		t.Fatal("Dependencies() must return a copy")
	}
}
