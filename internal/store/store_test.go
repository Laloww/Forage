package store

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Laloww/Forage/internal/chunk"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	return s
}

func makeChunks(texts ...string) []chunk.Chunk {
	chunks := make([]chunk.Chunk, len(texts))
	for i, text := range texts {
		chunks[i] = chunk.Chunk{
			ID:      "test_" + text[:min(5, len(text))],
			DocPath: "test.md",
			Text:    text,
			Index:   i,
		}
	}
	return chunks
}

func makeEmbeddings(vecs ...[]float32) [][]float32 {
	return vecs
}

func TestStore_AddAndCount(t *testing.T) {
	s := testStore(t)

	if s.Count() != 0 {
		t.Fatalf("new store should be empty, got %d", s.Count())
	}

	chunks := makeChunks("hello world", "goodbye world")
	embs := makeEmbeddings(
		[]float32{1, 0, 0},
		[]float32{0, 1, 0},
	)

	if err := s.Add(chunks, embs); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if s.Count() != 2 {
		t.Fatalf("expected 2 records, got %d", s.Count())
	}
}

func TestStore_AddMismatch(t *testing.T) {
	s := testStore(t)

	chunks := makeChunks("one", "two")
	embs := makeEmbeddings([]float32{1, 0})

	err := s.Add(chunks, embs)
	if err == nil {
		t.Fatal("expected error for mismatched counts")
	}
}

func TestStore_SearchVector(t *testing.T) {
	s := testStore(t)

	chunks := makeChunks("golang programming", "python scripting", "rust systems")
	embs := makeEmbeddings(
		[]float32{1, 0, 0},
		[]float32{0, 1, 0},
		[]float32{0, 0, 1},
	)

	if err := s.Add(chunks, embs); err != nil {
		t.Fatal(err)
	}

	// Query embedding close to first chunk
	results := s.Search("golang", []float32{0.9, 0.1, 0}, 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should be the golang chunk (closest vector)
	if results[0].Chunk.Text != "golang programming" {
		t.Errorf("expected 'golang programming' first, got %q", results[0].Chunk.Text)
	}
}

func TestStore_SearchBM25(t *testing.T) {
	s := testStore(t)

	chunks := makeChunks(
		"authentication with JWT tokens",
		"database connection pooling",
		"authentication middleware setup",
	)
	// All same embedding — so vector search gives no signal, BM25 decides
	emb := []float32{1, 0, 0}
	embs := makeEmbeddings(emb, emb, emb)

	if err := s.Add(chunks, embs); err != nil {
		t.Fatal(err)
	}

	results := s.Search("authentication", emb, 3)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// BM25 should rank auth chunks higher
	topTwo := results[0].Chunk.Text + " " + results[1].Chunk.Text
	if !(contains(topTwo, "authentication")) {
		t.Errorf("expected auth chunks in top 2, got: %s and %s", results[0].Chunk.Text, results[1].Chunk.Text)
	}
}

func TestStore_SearchEmpty(t *testing.T) {
	s := testStore(t)
	results := s.Search("query", []float32{1, 0}, 5)
	if results != nil {
		t.Fatalf("expected nil results for empty store, got %d", len(results))
	}
}

func TestStore_SearchTopKLargerThanStore(t *testing.T) {
	s := testStore(t)

	chunks := makeChunks("only one")
	embs := makeEmbeddings([]float32{1, 0})

	if err := s.Add(chunks, embs); err != nil {
		t.Fatal(err)
	}

	results := s.Search("one", []float32{1, 0}, 100)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (capped), got %d", len(results))
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Create and populate store
	s1, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("persistent data")
	embs := makeEmbeddings([]float32{1, 0, 0})
	if err := s1.Add(chunks, embs); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "index.gob")); err != nil {
		t.Fatalf("index.gob not created: %v", err)
	}

	// Load from same dir
	s2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Count() != 1 {
		t.Fatalf("reloaded store expected 1 record, got %d", s2.Count())
	}

	results := s2.Search("persistent", []float32{1, 0, 0}, 1)
	if len(results) != 1 {
		t.Fatal("search on reloaded store returned no results")
	}
	if results[0].Chunk.Text != "persistent data" {
		t.Errorf("unexpected text after reload: %q", results[0].Chunk.Text)
	}
}

func TestStore_Clear(t *testing.T) {
	s := testStore(t)

	chunks := makeChunks("to be cleared")
	embs := makeEmbeddings([]float32{1, 0})
	if err := s.Add(chunks, embs); err != nil {
		t.Fatal(err)
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if s.Count() != 0 {
		t.Fatalf("expected 0 after clear, got %d", s.Count())
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float32
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, -1.0},
		{"similar", []float32{1, 1, 0}, []float32{1, 0, 0}, 0.7071},
		{"empty", []float32{}, []float32{}, 0},
		{"length mismatch", []float32{1}, []float32{1, 0}, 0},
		{"zero vector", []float32{0, 0}, []float32{1, 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(got-tt.want)) > 0.001 {
				t.Errorf("cosineSimilarity(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"Hello, World!", 2},
		{"привет мир", 2},
		{"go1.22", 2},
		{"", 0},
		{"---", 0},
		{"one", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenize(tt.input)
			if len(tokens) != tt.want {
				t.Errorf("tokenize(%q) = %v (len %d), want len %d", tt.input, tokens, len(tokens), tt.want)
			}
		})
	}
}

func TestTokenize_CaseInsensitive(t *testing.T) {
	tokens := tokenize("Hello WORLD")
	if tokens[0] != "hello" || tokens[1] != "world" {
		t.Errorf("expected lowercase tokens, got %v", tokens)
	}
}

func TestRankIndices(t *testing.T) {
	scores := []float32{0.3, 0.9, 0.1, 0.7}
	ranks := rankIndices(scores)

	// 0.9 is rank 1, 0.7 is rank 2, 0.3 is rank 3, 0.1 is rank 4
	if ranks[1] != 1 {
		t.Errorf("expected rank 1 for score 0.9, got %d", ranks[1])
	}
	if ranks[3] != 2 {
		t.Errorf("expected rank 2 for score 0.7, got %d", ranks[3])
	}
	if ranks[0] != 3 {
		t.Errorf("expected rank 3 for score 0.3, got %d", ranks[0])
	}
	if ranks[2] != 4 {
		t.Errorf("expected rank 4 for score 0.1, got %d", ranks[2])
	}
}

func TestScoreBM25_EmptyAvgDL(t *testing.T) {
	s := testStore(t)
	// avgDL is 0 on empty store — should not panic
	scores := s.scoreBM25("test query")
	if len(scores) != 0 {
		t.Fatalf("expected empty scores, got %d", len(scores))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
