package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/model"
	"github.com/harunnryd/heike/internal/store"
)

const (
	CollectionMemory = "memories"
)

type VectorMemory struct {
	store          *store.Worker
	router         model.ModelRouter
	embeddingModel string
}

func NewManager(s *store.Worker, r model.ModelRouter, embeddingModel string) *VectorMemory {
	embeddingModel = strings.TrimSpace(embeddingModel)
	if embeddingModel == "" {
		embeddingModel = config.DefaultModelEmbedding
	}

	return &VectorMemory{
		store:          s,
		router:         r,
		embeddingModel: embeddingModel,
	}
}

// Ensure VectorMemory implements cognitive.MemoryManager
var _ cognitive.MemoryManager = (*VectorMemory)(nil)

func (m *VectorMemory) Retrieve(ctx context.Context, query string) ([]string, error) {
	embedding, err := m.router.RouteEmbedding(ctx, m.embeddingModel, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	results, err := m.store.SearchVectors(CollectionMemory, embedding, 5) // Top 5
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	var facts []string
	for _, r := range results {
		facts = append(facts, r.Content)
	}

	slog.Info("Memory retrieved", "query", query, "count", len(facts))
	return facts, nil
}

func (m *VectorMemory) Remember(ctx context.Context, fact string) error {
	embedding, err := m.router.RouteEmbedding(ctx, m.embeddingModel, fact)
	if err != nil {
		return fmt.Errorf("failed to embed fact: %w", err)
	}

	id := ulid.Make().String()

	// Metadata can be empty for now
	err = m.store.UpsertVector(CollectionMemory, id, embedding, nil, fact)
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	slog.Info("Memory stored", "fact_preview", fact[:min(len(fact), 50)], "id", id)
	return nil
}
