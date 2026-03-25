# @lalow123/forage

**fo*RAG*e through your docs. Offline. Fast. Zero dependencies.**

Local-first RAG search tool with hybrid BM25 + vector search. Written in Go, distributed as a single binary. Works as an MCP server for Claude Code, Cursor, and other AI assistants.

## Install into Claude Code

```bash
claude mcp add forage -- npx @lalow123/forage
```

That's it. Claude now has access to `forage_search`, `forage_index`, and `forage_stats` tools.

## How it works

1. **Index** your local files (docs, code, configs)
2. **Search** using hybrid semantic + keyword matching
3. Embeddings run locally via [Ollama](https://ollama.com) — your data never leaves your machine

## Prerequisites

Install [Ollama](https://ollama.com/download) and pull an embedding model:

```bash
ollama pull nomic-embed-text
```

## CLI usage

You can also use forage as a standalone CLI:

```bash
npx @lalow123/forage index ./docs
npx @lalow123/forage search "how does auth work"
npx @lalow123/forage stats
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `forage_search` | Search indexed documents. Params: `query`, `top_k` |
| `forage_index` | Index files from a directory. Params: `path` |
| `forage_stats` | Show index statistics |

## Search features

- **Hybrid search** — BM25 keyword + vector cosine similarity
- **RRF fusion** — Reciprocal Rank Fusion combines both methods for better results
- **17+ file types** — `.md`, `.txt`, `.go`, `.py`, `.js`, `.ts`, `.rs`, `.yaml`, `.json`, and more
- **Multilingual** — English, Russian, Chinese, and any language Ollama supports
- **Fast** — sub-millisecond search, Go concurrency for embedding generation

## Links

- [GitHub](https://github.com/Laloww/Forage) — full documentation, architecture, API reference
- [Releases](https://github.com/Laloww/Forage/releases) — pre-built binaries

## License

MIT
