# LinkedIn Post — forage launch

---

## English (recommended for reach)

I got tired of Python dependency hell just to search my own docs.

So I built forage — a local-first RAG search tool written in Go.

One binary. No Python. No cloud. Your data never leaves your machine.

How it works:
→ forage index ./docs — indexes your files into a local vector store
→ forage search "how does auth work" — hybrid search in <1ms

What makes it different:
• Pure Go, zero external dependencies — go install and done
• Hybrid search: BM25 keywords + vector cosine similarity, fused with Reciprocal Rank Fusion
• Works with Ollama locally — fully offline, fully private
• Supports .md, .txt, .go, .py, .ts, .rs and 10+ more file types
• Built-in HTTP API for integrations
• 92% test coverage

The Go RAG ecosystem is almost empty.
Most AI tooling is Python-first. But Go gives you:
— Single binary distribution
— No runtime dependencies
— Native concurrency for fast embedding generation

And yes, the name is a wordplay — fo*RAG*e.

Open source, MIT licensed.

GitHub: https://github.com/Laloww/Forage

What would you add to a local-first RAG tool?

#opensource #golang #rag #ai #llm #search

---

## Русский

Надоело ставить Python и 50 зависимостей ради поиска по своим документам.

Написал forage — локальный RAG-поиск на Go.

Один бинарник. Без Python. Без облака. Данные не покидают машину.

Как работает:
→ forage index ./docs — индексирует файлы в локальный векторный стор
→ forage search "как настроить авторизацию" — гибридный поиск за <1мс

Что под капотом:
• Pure Go, ноль внешних зависимостей
• Гибридный поиск: BM25 (ключевые слова) + cosine similarity (семантика), через Reciprocal Rank Fusion
• Эмбеддинги через Ollama — полностью офлайн
• 17+ типов файлов: .md, .txt, .go, .py, .ts и другие
• Встроенный HTTP API
• 92% тестовое покрытие

И да, RAG спрятан в названии — fo*RAG*e.

Open source, MIT.

GitHub: https://github.com/Laloww/Forage

Что бы вы добавили?

#opensource #golang #rag #ai #llm

---

## Tips

1. Post Tue-Thu, 8-11 AM local time
2. Reply to every comment in the first 2 hours
3. Follow-up post in 1 week with results/stars
4. Cross-post: Hacker News (Show HN), r/golang, r/LocalLLaMA
