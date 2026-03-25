package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Laloww/Forage/internal/chunk"
	"github.com/Laloww/Forage/internal/embed"
	"github.com/Laloww/Forage/internal/loader"
	"github.com/Laloww/Forage/internal/store"
)

// Server implements the MCP protocol over stdio (JSON-RPC 2.0).
type Server struct {
	storeDir  string
	ollamaURL string
	model     string
}

// New creates an MCP server.
func New(storeDir, ollamaURL, model string) *Server {
	return &Server{
		storeDir:  storeDir,
		ollamaURL: ollamaURL,
		model:     model,
	}
}

// Run starts the MCP server, reading JSON-RPC from stdin and writing to stdout.
func (s *Server) Run() error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		line, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeResponse(writer, errorResponse(nil, -32700, "Parse error"))
			continue
		}

		resp := s.handleRequest(req)
		writeResponse(writer, resp)
	}
}

func (s *Server) handleRequest(req jsonRPCRequest) jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return jsonRPCResponse{} // no response for notifications
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return errorResponse(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req jsonRPCRequest) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "forage",
				"version": "0.1.0",
			},
		},
	}
}

func (s *Server) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
	tools := []map[string]any{
		{
			"name":        "forage_search",
			"description": "Search indexed documents using hybrid semantic + keyword search (BM25 + cosine + RRF fusion). Returns the most relevant text chunks.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query text",
					},
					"top_k": map[string]any{
						"type":        "number",
						"description": "Number of results to return (default: 5, max: 20)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "forage_index",
			"description": "Index files from a directory. Loads supported file types (.md, .txt, .go, .py, .ts, etc.), splits into chunks, generates embeddings via Ollama, and stores in the local vector store.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Directory path to index",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "forage_stats",
			"description": "Show index statistics: number of indexed chunks and store location.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(req jsonRPCRequest) jsonRPCResponse {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}

	data, err := json.Marshal(req.Params)
	if err != nil {
		return errorResponse(req.ID, -32602, "Invalid params")
	}
	if err := json.Unmarshal(data, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params")
	}

	switch params.Name {
	case "forage_search":
		return s.toolSearch(req.ID, params.Arguments)
	case "forage_index":
		return s.toolIndex(req.ID, params.Arguments)
	case "forage_stats":
		return s.toolStats(req.ID)
	default:
		return toolError(req.ID, "Unknown tool: "+params.Name)
	}
}

func (s *Server) toolSearch(id any, args map[string]any) jsonRPCResponse {
	query, _ := args["query"].(string)
	if query == "" {
		return toolError(id, "query is required")
	}

	topK := 5
	if v, ok := args["top_k"].(float64); ok && v > 0 {
		topK = int(v)
		if topK > 20 {
			topK = 20
		}
	}

	st, err := store.New(s.storeDir)
	if err != nil {
		return toolError(id, "store error: "+err.Error())
	}

	if st.Count() == 0 {
		return toolError(id, "Index is empty. Use forage_index first.")
	}

	embedder := embed.New(s.ollamaURL, s.model)
	queryEmb, err := embedder.EmbedSingle(query)
	if err != nil {
		return toolError(id, "embedding error: "+err.Error())
	}

	results := st.Search(query, queryEmb, topK)

	var sb strings.Builder
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("--- Result #%d (score: %.4f) from %s ---\n", i+1, r.Score, r.Chunk.DocPath))
		sb.WriteString(r.Chunk.Text)
		sb.WriteString("\n\n")
	}

	return toolResult(id, sb.String())
}

func (s *Server) toolIndex(id any, args map[string]any) jsonRPCResponse {
	path, _ := args["path"].(string)
	if path == "" {
		return toolError(id, "path is required")
	}

	docs, err := loader.LoadDir(path)
	if err != nil {
		return toolError(id, "load error: "+err.Error())
	}
	if len(docs) == 0 {
		return toolError(id, "no supported files found in "+path)
	}

	chunks := chunk.Split(docs, chunk.DefaultOptions())

	embedder := embed.New(s.ollamaURL, s.model)

	const batchSize = 32
	var allEmbeddings [][]float32
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		texts := make([]string, end-i)
		for j, c := range chunks[i:end] {
			texts[j] = c.Text
		}

		embs, err := embedder.Embed(texts)
		if err != nil {
			return toolError(id, "embedding error: "+err.Error())
		}
		allEmbeddings = append(allEmbeddings, embs...)
	}

	st, err := store.New(s.storeDir)
	if err != nil {
		return toolError(id, "store error: "+err.Error())
	}

	if err := st.Add(chunks, allEmbeddings); err != nil {
		return toolError(id, "store add error: "+err.Error())
	}

	msg := fmt.Sprintf("Indexed %d chunks from %d files in %s", len(chunks), len(docs), path)
	return toolResult(id, msg)
}

func (s *Server) toolStats(id any) jsonRPCResponse {
	st, err := store.New(s.storeDir)
	if err != nil {
		return toolError(id, "store error: "+err.Error())
	}

	msg := "Chunks indexed: " + strconv.Itoa(st.Count()) + "\nStore: " + s.storeDir + "/"
	return toolResult(id, msg)
}

// --- JSON-RPC types ---

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc,omitempty"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// --- Helpers ---

func readMessage(reader *bufio.Reader) ([]byte, error) {
	// Read Content-Length header
	var contentLength int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(val)
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}

	return body, nil
}

func writeResponse(w io.Writer, resp jsonRPCResponse) {
	if resp.JSONRPC == "" {
		return // notification, no response
	}
	data, _ := json.Marshal(resp)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	fmt.Fprint(w, header)
	w.Write(data)
}

func errorResponse(id any, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

func toolResult(id any, text string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": text,
				},
			},
		},
	}
}

func toolError(id any, text string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": "Error: " + text,
				},
			},
			"isError": true,
		},
	}
}
