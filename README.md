# forage

**fo*RAG*e through your docs. Offline. Fast. Zero dependencies.**

forage indexes your local files and lets you search them with hybrid semantic + keyword search — all running locally on your machine.

```
forage index ./docs
forage search "how to configure authentication"
```

## Why forage?

- **Zero Python dependencies** — one Go binary, `go install` and done
- **Fully local** — embeddings via [Ollama](https://ollama.com), data never leaves your machine
- **Hybrid search** — BM25 keyword + vector cosine similarity, fused with Reciprocal Rank Fusion (RRF)
- **Fast** — Go concurrency, in-memory search with disk persistence
- **Multilingual** — works with English, Russian, and any language your embedding model supports

## Quick Start

### 1. Install

```bash
go install github.com/Laloww/Forage/cmd/forage@latest
```

Or build from source:

```bash
git clone https://github.com/Laloww/Forage.git
cd Forage
go build -o forage ./cmd/forage/
```

### 2. Start Ollama

```bash
# Install: https://ollama.com
ollama pull nomic-embed-text
```

### 3. Index your files

```bash
forage index ./docs
# 📂 Loading files from ./docs...
#    Found 42 files
# ✂️  Split into 156 chunks
# 🧠 Generating embeddings via nomic-embed-text...
#    156/156 chunks embedded (3.2s)
# ✅ Indexed 156 chunks from 42 files in 3.2s
```

### 4. Search

```bash
forage search "how does authentication work"
# 🔍 5 results (0.8ms)
#
# ─── #1  score: 0.0323  📄 docs/auth.md ───
# Authentication is handled via JWT tokens...
```

### 5. HTTP API (optional)

```bash
forage serve --port 8080
```

```bash
curl -X POST localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{"query": "auth config", "top_k": 5}'
```

### 6. Claude Code / MCP integration

```bash
# One command to add forage to Claude Code:
claude mcp add forage -- npx @lalow123/forage

# Or if you have the binary:
claude mcp add forage -- forage mcp
```

Now Claude can search your indexed docs directly during conversations.

## Supported File Types

`.md` `.txt` `.go` `.py` `.js` `.ts` `.rs` `.yaml` `.yml` `.json` `.toml` `.html` `.css` `.xml` `.csv` `.sh` `.log`

## How It Works

```
Documents → Chunking → Embeddings (Ollama) → Vector Store
                                                   ↓
Query → Embedding + Tokenization → Hybrid Search (BM25 + Cosine + RRF)
                                                   ↓
                                            Ranked Results
```

1. **Chunking** — documents are split into overlapping paragraphs (~512 chars)
2. **Embedding** — each chunk is embedded via Ollama (`nomic-embed-text`)
3. **Indexing** — embeddings + text stored in a pure-Go vector store
4. **Search** — query goes through both:
   - **BM25** (keyword matching with TF-IDF)
   - **Cosine similarity** (semantic matching)
5. **Fusion** — results from both methods are combined using **Reciprocal Rank Fusion** (RRF), giving you the best of both worlds

## Options

```
--ollama-url <url>     Ollama URL (default: http://localhost:11434)
--model <name>         Embedding model (default: nomic-embed-text)
--top-k <n>            Number of results (default: 5)
--chunk-size <n>       Chunk size in chars (default: 512)
--port <n>             Server port (default: 8080)
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/search` | Hybrid search. Body: `{"query": "...", "top_k": 5}` |
| `GET` | `/stats` | Index statistics |
| `GET` | `/health` | Health check |

## Architecture

```
forage/
├── cmd/forage/         # CLI entrypoint
└── internal/
    ├── loader/         # File discovery and loading
    ├── chunk/          # Text chunking with overlap
    ├── embed/          # Ollama embedding client
    ├── store/          # Vector store + BM25 + RRF fusion
    └── server/         # HTTP API
```

**No external Go dependencies.** Pure standard library.

## Testing

```bash
go test ./... -cover
# 38 tests, 92.6% coverage
```

## Roadmap

- [ ] PDF and DOCX support
- [ ] Watch mode (`forage watch ./docs`) — re-index on file changes
- [ ] MCP server for Claude Code / Cursor integration
- [ ] Graph RAG — document-level knowledge graph
- [ ] Reranking with cross-encoder models
- [ ] OpenAI API compatibility for embeddings
- [ ] Incremental indexing (only changed files)

## License

MIT
