package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Laloww/Forage/internal/chunk"
	"github.com/Laloww/Forage/internal/embed"
	"github.com/Laloww/Forage/internal/store"
)

func mockEmbedServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		embeddings := make([][]float32, len(req.Input))
		for i := range req.Input {
			embeddings[i] = []float32{0.1, 0.2, 0.3}
		}
		json.NewEncoder(w).Encode(map[string]any{"embeddings": embeddings})
	}))
}

func setupTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()

	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}

	ollamaSrv := mockEmbedServer(t)
	t.Cleanup(ollamaSrv.Close)

	e := embed.New(ollamaSrv.URL, "test-model")
	srv := New(s, e)

	// Add some data
	chunks := []chunk.Chunk{
		{ID: "1", DocPath: "auth.md", Text: "JWT authentication tokens", Index: 0},
		{ID: "2", DocPath: "db.md", Text: "Database connection pooling", Index: 0},
		{ID: "3", DocPath: "api.md", Text: "REST API endpoints", Index: 0},
	}
	embs := [][]float32{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	if err := s.Add(chunks, embs); err != nil {
		t.Fatal(err)
	}

	return srv, s
}

func TestHealth(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestStats(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	chunks, ok := resp["chunks"].(float64)
	if !ok || chunks != 3 {
		t.Errorf("expected 3 chunks, got %v", resp["chunks"])
	}
}

func TestSearch_Success(t *testing.T) {
	srv, _ := setupTestServer(t)

	body := `{"query": "authentication", "top_k": 2}`
	req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp searchResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.Took == "" {
		t.Error("expected non-empty took field")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	srv, _ := setupTestServer(t)

	body := `{"query": "", "top_k": 5}`
	req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty query, got %d", w.Code)
	}
}

func TestSearch_InvalidJSON(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/search", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestSearch_DefaultTopK(t *testing.T) {
	srv, _ := setupTestServer(t)

	body := `{"query": "test"}`
	req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp searchResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Default top_k=5, but only 3 records in store
	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results (capped by store size), got %d", len(resp.Results))
	}
}

func TestSearch_TopKCapped(t *testing.T) {
	srv, _ := setupTestServer(t)

	body := `{"query": "test", "top_k": 999}`
	req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp searchResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// top_k > maxTopK(100) → reset to 5, but only 3 in store
	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}
}
