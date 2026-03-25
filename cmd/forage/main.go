package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Laloww/Forage/internal/chunk"
	"github.com/Laloww/Forage/internal/embed"
	"github.com/Laloww/Forage/internal/loader"
	"github.com/Laloww/Forage/internal/mcp"
	"github.com/Laloww/Forage/internal/server"
	"github.com/Laloww/Forage/internal/store"
)

const (
	version    = "0.2.0"
	storeDir   = ".forage"
	batchSize  = 32
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		cmdIndex()
	case "search":
		cmdSearch()
	case "serve":
		cmdServe()
	case "stats":
		cmdStats()
	case "clear":
		cmdClear()
	case "mcp":
		cmdMCP()
	case "version":
		fmt.Printf("forage v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`forage — local-first RAG search. No Python, single binary.

Usage:
  forage index <path>           Index files from directory
  forage search <query>         Semantic + keyword hybrid search
  forage serve [--port 8080]    Start HTTP API server
  forage mcp                    Start MCP server (for Claude Code)
  forage stats                  Show index statistics
  forage clear                  Remove all indexed data
  forage version                Print version

Options:
  --ollama-url <url>           Ollama URL (default: http://localhost:11434)
  --model <name>               Embedding model (default: nomic-embed-text)
  --top-k <n>                  Number of results (default: 5)
  --chunk-size <n>             Chunk size in chars (default: 512)

Examples:
  forage index ./docs
  forage search "how to configure auth"
  forage search "настройка авторизации" --top-k 3
  curl -X POST localhost:8080/search -d '{"query":"auth","top_k":5}'
`)
}

// --- Commands ---

func cmdIndex() {
	if len(os.Args) < 3 {
		fatal("usage: forage index <path>")
	}

	path := os.Args[2]
	opts := parseFlags()

	fmt.Printf("📂 Loading files from %s...\n", path)
	docs, err := loader.LoadDir(path)
	if err != nil {
		fatal("load error: %v", err)
	}
	if len(docs) == 0 {
		fatal("no supported files found in %s", path)
	}
	fmt.Printf("   Found %d files\n", len(docs))

	chunkOpts := chunk.DefaultOptions()
	if opts.chunkSize > 0 {
		chunkOpts.Size = opts.chunkSize
	}
	chunks := chunk.Split(docs, chunkOpts)
	fmt.Printf("✂️  Split into %d chunks\n", len(chunks))

	embedder := embed.New(opts.ollamaURL, opts.model)

	fmt.Printf("🧠 Generating embeddings via %s...\n", opts.model)
	start := time.Now()

	var allEmbeddings [][]float32
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		texts := make([]string, len(batch))
		for j, c := range batch {
			texts[j] = c.Text
		}

		embs, err := embedder.Embed(texts)
		if err != nil {
			fatal("embedding error: %v", err)
		}
		allEmbeddings = append(allEmbeddings, embs...)

		fmt.Printf("   %d/%d chunks embedded\r", len(allEmbeddings), len(chunks))
	}
	fmt.Printf("   %d/%d chunks embedded (%.1fs)\n", len(allEmbeddings), len(chunks), time.Since(start).Seconds())

	s, err := store.New(storeDir)
	if err != nil {
		fatal("store error: %v", err)
	}

	if err := s.Add(chunks, allEmbeddings); err != nil {
		fatal("store add error: %v", err)
	}

	fmt.Printf("✅ Indexed %d chunks from %d files in %s\n", len(chunks), len(docs), time.Since(start).Round(time.Millisecond))
}

func cmdSearch() {
	if len(os.Args) < 3 {
		fatal("usage: forage search <query>")
	}

	query := os.Args[2]
	opts := parseFlags()

	s, err := store.New(storeDir)
	if err != nil {
		fatal("store error: %v", err)
	}

	if s.Count() == 0 {
		fatal("index is empty — run 'forage index <path>' first")
	}

	embedder := embed.New(opts.ollamaURL, opts.model)
	queryEmb, err := embedder.EmbedSingle(query)
	if err != nil {
		fatal("embedding error: %v", err)
	}

	start := time.Now()
	results := s.Search(query, queryEmb, opts.topK)
	took := time.Since(start)

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	fmt.Printf("🔍 %d results (%.1fms)\n\n", len(results), float64(took.Microseconds())/1000)

	for i, r := range results {
		fmt.Printf("─── #%d  score: %.4f  📄 %s ───\n", i+1, r.Score, r.Chunk.DocPath)
		text := r.Chunk.Text
		if len(text) > 300 {
			text = text[:300] + "..."
		}
		fmt.Printf("%s\n\n", text)
	}
}

func cmdServe() {
	opts := parseFlags()

	s, err := store.New(storeDir)
	if err != nil {
		fatal("store error: %v", err)
	}

	if s.Count() == 0 {
		fmt.Println("⚠️  Index is empty. Run 'forage index <path>' first.")
	}

	embedder := embed.New(opts.ollamaURL, opts.model)
	srv := server.New(s, embedder)

	addr := fmt.Sprintf(":%d", opts.port)
	fmt.Printf("🚀 forage server listening on http://localhost%s\n", addr)
	fmt.Printf("   POST /search  — hybrid search\n")
	fmt.Printf("   GET  /stats   — index stats\n")
	fmt.Printf("   GET  /health  — health check\n")

	if err := srv.ListenAndServe(addr); err != nil {
		fatal("server error: %v", err)
	}
}

func cmdMCP() {
	opts := parseFlags()
	srv := mcp.New(storeDir, opts.ollamaURL, opts.model)
	if err := srv.Run(); err != nil {
		fatal("mcp server error: %v", err)
	}
}

func cmdStats() {
	s, err := store.New(storeDir)
	if err != nil {
		fatal("store error: %v", err)
	}

	fmt.Printf("📊 Index: %d chunks\n", s.Count())
	fmt.Printf("📁 Store: %s/\n", storeDir)
}

func cmdClear() {
	s, err := store.New(storeDir)
	if err != nil {
		fatal("store error: %v", err)
	}
	if err := s.Clear(); err != nil && !os.IsNotExist(err) {
		fatal("clear error: %v", err)
	}
	fmt.Println("🗑️  Index cleared.")
}

// --- Flag parsing ---

type flags struct {
	ollamaURL string
	model     string
	topK      int
	port      int
	chunkSize int
}

func parseFlags() flags {
	f := flags{
		ollamaURL: "http://localhost:11434",
		model:     "nomic-embed-text",
		topK:      5,
		port:      8080,
		chunkSize: 512,
	}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ollama-url":
			if i+1 < len(args) {
				f.ollamaURL = args[i+1]
				i++
			}
		case "--model":
			if i+1 < len(args) {
				f.model = args[i+1]
				i++
			}
		case "--top-k":
			if i+1 < len(args) {
				f.topK = parseIntFlag(args[i+1], 5)
				i++
			}
		case "--port":
			if i+1 < len(args) {
				f.port = parseIntFlag(args[i+1], 8080)
				i++
			}
		case "--chunk-size":
			if i+1 < len(args) {
				f.chunkSize = parseIntFlag(args[i+1], 512)
				i++
			}
		}
	}
	return f
}

func parseIntFlag(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func fatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)
	os.Exit(1)
}
