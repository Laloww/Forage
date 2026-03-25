package store

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Laloww/Forage/internal/chunk"
)

// Record is a stored chunk with its embedding.
type Record struct {
	Chunk     chunk.Chunk
	Embedding []float32
}

// Result is a search result with score.
type Result struct {
	Chunk chunk.Chunk
	Score float32
}

// Store is an in-memory vector store with disk persistence.
type Store struct {
	mu      sync.RWMutex
	records []Record
	path    string

	// BM25 inverted index
	idf     map[string]float64
	termFreq []map[string]int
	docLens  []int
	avgDL    float64
}

// New creates or loads a store from the given directory.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	s := &Store{
		path: filepath.Join(dir, "index.gob"),
		idf:  make(map[string]float64),
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load store: %w", err)
	}

	return s, nil
}

// Add inserts records into the store and persists to disk.
func (s *Store) Add(chunks []chunk.Chunk, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks/embeddings count mismatch: %d vs %d", len(chunks), len(embeddings))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range chunks {
		s.records = append(s.records, Record{
			Chunk:     c,
			Embedding: embeddings[i],
		})
	}

	s.buildBM25Index()
	return s.save()
}

// Search performs hybrid search: cosine similarity + BM25, fused with RRF.
func (s *Store) Search(query string, queryEmb []float32, topK int) []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.records) == 0 {
		return nil
	}

	n := len(s.records)

	// Vector search: cosine similarity
	vectorScores := make([]float32, n)
	for i, rec := range s.records {
		vectorScores[i] = cosineSimilarity(queryEmb, rec.Embedding)
	}

	// BM25 search
	bm25Scores := s.scoreBM25(query)

	// Rank by each method
	vectorRanks := rankIndices(vectorScores)
	bm25Ranks := rankIndices(bm25Scores)

	// RRF fusion (k=60)
	const k = 60.0
	fusedScores := make([]float32, n)
	for i := range s.records {
		fusedScores[i] = float32(1.0/(k+float64(vectorRanks[i])) + 1.0/(k+float64(bm25Ranks[i])))
	}

	// Sort by fused score descending
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return fusedScores[indices[a]] > fusedScores[indices[b]]
	})

	if topK > n {
		topK = n
	}

	results := make([]Result, topK)
	for i := 0; i < topK; i++ {
		idx := indices[i]
		results[i] = Result{
			Chunk: s.records[idx].Chunk,
			Score: fusedScores[idx],
		}
	}

	return results
}

// Count returns the number of stored records.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// Clear removes all records and deletes the persisted file.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = nil
	s.idf = make(map[string]float64)
	s.termFreq = nil
	s.docLens = nil
	s.avgDL = 0

	return os.Remove(s.path)
}

// --- BM25 ---

func (s *Store) buildBM25Index() {
	n := len(s.records)
	s.termFreq = make([]map[string]int, n)
	s.docLens = make([]int, n)
	df := make(map[string]int) // document frequency

	totalLen := 0
	for i, rec := range s.records {
		tokens := tokenize(rec.Chunk.Text)
		s.docLens[i] = len(tokens)
		totalLen += len(tokens)

		tf := make(map[string]int)
		seen := make(map[string]bool)
		for _, t := range tokens {
			tf[t]++
			if !seen[t] {
				df[t]++
				seen[t] = true
			}
		}
		s.termFreq[i] = tf
	}

	if n > 0 {
		s.avgDL = float64(totalLen) / float64(n)
	}

	s.idf = make(map[string]float64)
	for term, freq := range df {
		s.idf[term] = math.Log(1 + (float64(n)-float64(freq)+0.5)/(float64(freq)+0.5))
	}
}

func (s *Store) scoreBM25(query string) []float32 {
	scores := make([]float32, len(s.records))
	if s.avgDL == 0 {
		return scores
	}

	tokens := tokenize(query)

	const (
		k1 = 1.2
		b  = 0.75
	)

	for i := range s.records {
		var score float64
		dl := float64(s.docLens[i])
		for _, t := range tokens {
			tf := float64(s.termFreq[i][t])
			idf := s.idf[t]
			num := tf * (k1 + 1)
			denom := tf + k1*(1-b+b*dl/s.avgDL)
			score += idf * num / denom
		}
		scores[i] = float32(score)
	}

	return scores
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 'а' && r <= 'я' || r == 'ё' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// --- Vector math ---

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / float32(math.Sqrt(float64(normA)*float64(normB)))
}

func rankIndices(scores []float32) []int {
	n := len(scores)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		return scores[indices[a]] > scores[indices[b]]
	})

	ranks := make([]int, n)
	for rank, idx := range indices {
		ranks[idx] = rank + 1
	}
	return ranks
}

// --- Persistence ---

func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "index-*.gob.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if err := gob.NewEncoder(tmp).Encode(s.records); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("encode records: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	return os.Rename(tmpName, s.path)
}

func (s *Store) load() error {
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := gob.NewDecoder(f).Decode(&s.records); err != nil {
		return err
	}

	s.buildBM25Index()
	return nil
}
