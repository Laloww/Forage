package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder generates vector embeddings from text.
type Embedder struct {
	baseURL string
	model   string
	client  *http.Client
}

// New creates an Embedder that talks to Ollama.
func New(baseURL, model string) *Embedder {
	return &Embedder{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// DefaultEmbedder returns an embedder using local Ollama with nomic-embed-text.
func DefaultEmbedder() *Embedder {
	return New("http://localhost:11434", "nomic-embed-text")
}

type embedRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed generates embeddings for a batch of texts.
func (e *Embedder) Embed(texts []string) ([][]float32, error) {
	reqBody := embedRequest{
		Model: e.model,
		Input: texts,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := e.client.Post(e.baseURL+"/api/embed", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}

	return result.Embeddings, nil
}

// EmbedSingle generates an embedding for a single text.
func (e *Embedder) EmbedSingle(text string) ([]float32, error) {
	results, err := e.Embed([]string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

// Model returns the model name.
func (e *Embedder) Model() string {
	return e.model
}
