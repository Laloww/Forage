package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Laloww/Forage/internal/embed"
	"github.com/Laloww/Forage/internal/store"
)

// Server provides an HTTP API for searching the index.
type Server struct {
	store    *store.Store
	embedder *embed.Embedder
	mux      *http.ServeMux
}

// New creates a new HTTP server.
func New(s *store.Store, e *embed.Embedder) *Server {
	srv := &Server{
		store:    s,
		embedder: e,
		mux:      http.NewServeMux(),
	}

	srv.mux.HandleFunc("GET /health", srv.handleHealth)
	srv.mux.HandleFunc("GET /stats", srv.handleStats)
	srv.mux.HandleFunc("POST /search", srv.handleSearch)

	return srv
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return httpSrv.ListenAndServe()
}

type searchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

type searchResultItem struct {
	Text    string  `json:"text"`
	DocPath string  `json:"doc_path"`
	Score   float32 `json:"score"`
}

type searchResponse struct {
	Results []searchResultItem `json:"results"`
	Took    string             `json:"took"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"chunks": s.store.Count(),
		"model":  s.embedder.Model(),
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit

	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query is required"})
		return
	}

	const maxTopK = 100
	if req.TopK <= 0 || req.TopK > maxTopK {
		req.TopK = 5
	}

	start := time.Now()

	queryEmb, err := s.embedder.EmbedSingle(req.Query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("embedding failed: %v", err),
		})
		return
	}

	results := s.store.Search(req.Query, queryEmb, req.TopK)

	items := make([]searchResultItem, len(results))
	for i, res := range results {
		items[i] = searchResultItem{
			Text:    res.Chunk.Text,
			DocPath: res.Chunk.DocPath,
			Score:   res.Score,
		}
	}

	writeJSON(w, http.StatusOK, searchResponse{
		Results: items,
		Took:    time.Since(start).String(),
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "writeJSON: %v\n", err)
	}
}
