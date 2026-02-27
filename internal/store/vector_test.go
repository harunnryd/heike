package store

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorOps(t *testing.T) {
	// Setup temporary home directory
	tmpHome, err := os.MkdirTemp("", "heike_vector_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	// Initialize Worker
	w, err := NewWorker("test-vector-ws", "", RuntimeConfig{})
	require.NoError(t, err)
	w.Start()
	defer w.Stop()

	// Test Data
	collection := "test_memory"
	id := "mem_01"
	vector := []float32{0.1, 0.2, 0.3, 0.4, 0.5} // 5D vector
	metadata := map[string]string{"source": "test", "type": "fact"}
	content := "The sky is blue."

	// Upsert
	err = w.UpsertVector(collection, id, vector, metadata, content)
	require.NoError(t, err)

	// Search (Exact Match)
	results, err := w.SearchVectors(collection, vector, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, id, results[0].ID)
	assert.Equal(t, content, results[0].Content)
	assert.Equal(t, metadata, results[0].Metadata)
	// Similarity should be close to 1.0 (cosine similarity of same vector)
	assert.InDelta(t, 1.0, results[0].Score, 0.0001)

	// Search (Different Vector)
	// Orthogonal-ish vector
	diffVector := []float32{0.9, 0.1, 0.0, 0.0, 0.0}
	results, err = w.SearchVectors(collection, diffVector, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, id, results[0].ID)
	assert.Less(t, results[0].Score, float32(0.9)) // Should be lower score
}
