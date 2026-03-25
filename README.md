<p align="center">
  <h1 align="center">forage</h1>
  <p align="center"><strong>fo<em>RAG</em>e through your docs. Offline. Fast. Zero dependencies.</strong></p>
  <p align="center">
    <a href="https://github.com/Laloww/Forage/actions"><img src="https://github.com/Laloww/Forage/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/Laloww/Forage/releases"><img src="https://img.shields.io/github/v/release/Laloww/Forage" alt="Release"></a>
    <a href="https://www.npmjs.com/package/@lalow123/forage"><img src="https://img.shields.io/npm/v/@lalow123/forage" alt="npm"></a>
    <a href="https://goreportcard.com/report/github.com/Laloww/Forage"><img src="https://goreportcard.com/badge/github.com/Laloww/Forage" alt="Go Report Card"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="License"></a>
  </p>
</p>

---

**forage** is a local-first RAG (Retrieval-Augmented Generation) search tool written in pure Go. It indexes your local files — code, docs, configs — and lets you search them using hybrid semantic + keyword search. Everything runs on your machine: no cloud, no Python, no external dependencies.

```bash
forage index ./docs
forage search "how to configure authentication"
```

> **MCP ready** — works as a tool inside Claude Code, Cursor, and any MCP-compatible AI assistant.

---

## Why forage?

Most RAG tools require Python, a dozen pip packages, and a vector database running in Docker. forage takes a different approach:

| | forage | Typical RAG stack |
|---|---|---|
| **Install** | Single binary or `npx` | Python + pip + Docker |
| **Dependencies** | Zero (pure Go stdlib) | langchain, chromadb, sentence-transformers... |
| **Data privacy** | 100% local, never leaves your machine | Often sends data to cloud APIs |
| **Search method** | Hybrid: BM25 + Vector + RRF fusion | Usually vector-only |
| **Startup time** | Instant | Seconds to minutes |
| **Binary size** | ~9 MB | 500MB+ with all deps |

### Key features

- **Hybrid search** — combines BM25 keyword matching with vector cosine similarity using Reciprocal Rank Fusion (RRF). Better results than vector-only search.
- **Fully local** — embeddings via [Ollama](https://ollama.com). Your data stays on your machine.
- **Pure Go, zero dependencies** — one binary, compiles in seconds, runs everywhere.
- **MCP server built-in** — plug into Claude Code or Cursor with one command.
- **HTTP API** — integrate with any tool via REST.
- **Multilingual** — works with English, Russian, Chinese, and any language your embedding model supports.
- **17+ file types** — `.md`, `.txt`, `.go`, `.py`, `.js`, `.ts`, `.rs`, `.yaml`, `.json`, `.toml`, `.html`, `.css`, `.xml`, `.csv`, `.sh`, `.log`, and more.

---

## Installation

### Option 1: Go install

```bash
go install github.com/Laloww/Forage/cmd/forage@latest
```

### Option 2: Download binary

Grab a pre-built binary from [Releases](https://github.com/Laloww/Forage/releases) for your platform (macOS, Linux, Windows).

### Option 3: Build from source

```bash
git clone https://github.com/Laloww/Forage.git
cd Forage
go build -o forage ./cmd/forage/
```

### Option 4: npx (for MCP / Claude Code)

```bash
npx @lalow123/forage
```

### Prerequisites

You need [Ollama](https://ollama.com) running locally for embeddings:

```bash
# Install Ollama: https://ollama.com/download
ollama pull nomic-embed-text
```

---

## Quick Start

### 1. Index your files

```bash
forage index ./docs
```

```
📂 Loading files from ./docs...
   Found 42 files
✂️  Split into 156 chunks
🧠 Generating embeddings via nomic-embed-text...
   156/156 chunks embedded (3.2s)
✅ Indexed 156 chunks from 42 files in 3.2s
```

### 2. Search

```bash
forage search "how does authentication work"
```

```
🔍 5 results (0.8ms)

─── #1  score: 0.0323  📄 docs/auth.md ───
Authentication is handled via JWT tokens. The middleware validates
the token on each request and extracts the user context...

─── #2  score: 0.0301  📄 docs/api.md ───
All API endpoints require a valid Bearer token in the
Authorization header...
```

### 3. HTTP API

```bash
forage serve --port 8080
```

```bash
curl -X POST localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{"query": "auth config", "top_k": 5}'
```

Response:

```json
{
  "results": [
    {
      "text": "Authentication is handled via JWT tokens...",
      "doc_path": "docs/auth.md",
      "score": 0.0323
    }
  ],
  "took": "0.8ms"
}
```

### 4. Claude Code / MCP

Add forage as a tool for Claude Code with one command:

```bash
claude mcp add forage -- npx @lalow123/forage
```

Or if you have the binary installed:

```bash
claude mcp add forage -- forage mcp
```

Claude will now have access to three tools:
- **forage_search** — search your indexed documents
- **forage_index** — index a directory
- **forage_stats** — check index status

---

## How It Works

```
                          ┌─────────────┐
                          │  Documents   │
                          └──────┬──────┘
                                 │
                          ┌──────▼──────┐
                          │   Chunking   │  Split into ~512 char paragraphs
                          └──────┬──────┘
                                 │
                          ┌──────▼──────┐
                          │   Ollama     │  Generate embeddings
                          │  (local)     │  (nomic-embed-text)
                          └──────┬──────┘
                                 │
                          ┌──────▼──────┐
                          │ Vector Store │  Pure Go, persisted to disk
                          └──────┬──────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
  ┌──────▼──────┐         ┌──────▼──────┐         ┌──────▼──────┐
  │    BM25     │         │   Cosine    │         │     RRF     │
  │  (keyword)  │         │ (semantic)  │         │  (fusion)   │
  └──────┬──────┘         └──────┬──────┘         └──────┬──────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                          ┌──────▼──────┐
                          │   Ranked    │
                          │   Results   │
                          └─────────────┘
```

### Search pipeline

1. **Chunking** — documents are split into overlapping paragraphs (~512 chars). Overlap ensures context isn't lost at boundaries.

2. **Embedding** — each chunk is embedded using Ollama's `nomic-embed-text` model (768 dimensions). Runs fully local.

3. **Vector store** — embeddings are stored in a pure-Go in-memory store with gob-encoded disk persistence. No external database needed.

4. **Hybrid search** — every query goes through two parallel paths:
   - **BM25** — classic keyword matching with TF-IDF weighting. Great for exact terms.
   - **Cosine similarity** — semantic matching via vector embeddings. Great for meaning.

5. **RRF fusion** — results from both methods are combined using [Reciprocal Rank Fusion](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf) (k=60). This consistently outperforms either method alone.

---

## CLI Reference

```
forage index <path>           Index files from a directory
forage search <query>         Hybrid semantic + keyword search
forage serve [--port 8080]    Start HTTP API server
forage mcp                    Start MCP server (for Claude Code)
forage stats                  Show index statistics
forage clear                  Remove all indexed data
forage version                Print version
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--ollama-url` | `http://localhost:11434` | Ollama API URL |
| `--model` | `nomic-embed-text` | Embedding model name |
| `--top-k` | `5` | Number of search results |
| `--chunk-size` | `512` | Chunk size in characters |
| `--port` | `8080` | HTTP server port |

### Examples

```bash
# Index a project
forage index ./my-project

# Search with more results
forage search "database connection" --top-k 10

# Use a different embedding model
forage index ./docs --model mxbai-embed-large

# Start server on custom port
forage serve --port 3000

# Check what's indexed
forage stats

# Start fresh
forage clear
```

---

## HTTP API

### `POST /search`

Search the index.

```bash
curl -X POST localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{"query": "how to deploy", "top_k": 3}'
```

**Request body:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | yes | — | Search query |
| `top_k` | number | no | 5 | Max results (capped at 100) |

**Response:**

```json
{
  "results": [
    {
      "text": "Deploy using docker-compose up -d...",
      "doc_path": "docs/deploy.md",
      "score": 0.0312
    }
  ],
  "took": "1.2ms"
}
```

### `GET /stats`

```json
{
  "chunks": 156,
  "model": "nomic-embed-text"
}
```

### `GET /health`

```json
{
  "status": "ok"
}
```

---

## MCP Tools

When running as an MCP server (`forage mcp`), three tools are exposed:

| Tool | Description |
|------|-------------|
| `forage_search` | Search indexed documents. Params: `query` (required), `top_k` (optional, default 5) |
| `forage_index` | Index files from a path. Params: `path` (required) |
| `forage_stats` | Show index statistics. No params. |

---

## Architecture

```
forage/
├── cmd/forage/           # CLI entrypoint, flag parsing
├── internal/
│   ├── loader/           # Recursive file discovery, 17+ file types
│   ├── chunk/            # Paragraph-based chunking with overlap
│   ├── embed/            # Ollama HTTP client, batch embedding
│   ├── store/            # Vector store + BM25 index + RRF fusion
│   ├── server/           # REST API (net/http)
│   └── mcp/              # MCP server (JSON-RPC 2.0 over stdio)
├── npm/                  # npm package for npx distribution
├── .github/workflows/    # CI + release pipeline
└── .goreleaser.yml       # Cross-platform binary builds
```

**Pure Go standard library.** No external Go dependencies.

---

## Testing

```bash
go test ./... -cover
```

```
ok   internal/chunk     coverage: 100.0%
ok   internal/store     coverage: 93.7%
ok   internal/loader    coverage: 89.2%
ok   internal/embed     coverage: 87.5%
ok   internal/server    coverage: 85.3%
TOTAL                            92.6%
```

38 tests across all packages.

---

## Supported File Types

| Category | Extensions |
|----------|-----------|
| Documentation | `.md`, `.txt` |
| Code | `.go`, `.py`, `.js`, `.ts`, `.rs`, `.sh` |
| Config | `.yaml`, `.yml`, `.json`, `.toml`, `.cfg`, `.ini` |
| Web | `.html`, `.css`, `.xml` |
| Data | `.csv`, `.log` |

Automatically skips: `.git/`, `node_modules/`, `vendor/`, `__pycache__/`, `.env` files.

---

## Roadmap

- [ ] PDF and DOCX support
- [ ] Watch mode — `forage watch ./docs` to re-index on file changes
- [ ] Graph RAG — document-level knowledge graph with entity extraction
- [ ] Reranking with cross-encoder models
- [ ] OpenAI-compatible embedding API
- [ ] Incremental indexing (only changed files)
- [ ] Export results to JSON/CSV
- [ ] Cursor / Windsurf MCP integration guide

---

## Contributing

Contributions are welcome! Please open an issue first to discuss what you'd like to change.

```bash
git clone https://github.com/Laloww/Forage.git
cd Forage
go test ./...          # run tests
go build ./cmd/forage/ # build
```

---

## License

[MIT](LICENSE)
