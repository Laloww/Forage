package embed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockOllamaServer(t *testing.T, dim int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		embeddings := make([][]float32, len(req.Input))
		for i := range req.Input {
			emb := make([]float32, dim)
			for j := range emb {
				emb[j] = float32(i+1) * 0.1
			}
			embeddings[i] = emb
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"embeddings": embeddings,
		})
	}))
}

func TestEmbed_Single(t *testing.T) {
	srv := mockOllamaServer(t, 4)
	defer srv.Close()

	e := New(srv.URL, "test-model")
	emb, err := e.EmbedSingle("hello world")
	if err != nil {
		t.Fatalf("EmbedSingle: %v", err)
	}

	if len(emb) != 4 {
		t.Fatalf("expected 4-dim embedding, got %d", len(emb))
	}
}

func TestEmbed_Batch(t *testing.T) {
	srv := mockOllamaServer(t, 3)
	defer srv.Close()

	e := New(srv.URL, "test-model")
	texts := []string{"one", "two", "three"}
	embs, err := e.Embed(texts)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if len(embs) != 3 {
		t.Fatalf("expected 3 embeddings, got %d", len(embs))
	}

	for i, emb := range embs {
		if len(emb) != 3 {
			t.Errorf("embedding %d: expected 3 dims, got %d", i, len(emb))
		}
	}
}

func TestEmbed_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := New(srv.URL, "test-model")
	_, err := e.EmbedSingle("test")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestEmbed_ConnectionRefused(t *testing.T) {
	e := New("http://localhost:1", "test-model")
	_, err := e.EmbedSingle("test")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestEmbed_Model(t *testing.T) {
	e := New("http://localhost:11434", "nomic-embed-text")
	if e.Model() != "nomic-embed-text" {
		t.Errorf("expected model nomic-embed-text, got %s", e.Model())
	}
}

func TestDefaultEmbedder(t *testing.T) {
	e := DefaultEmbedder()
	if e.model != "nomic-embed-text" {
		t.Errorf("default model should be nomic-embed-text, got %s", e.model)
	}
	if e.baseURL != "http://localhost:11434" {
		t.Errorf("default URL should be localhost:11434, got %s", e.baseURL)
	}
}
