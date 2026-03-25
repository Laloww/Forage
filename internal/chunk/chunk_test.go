package chunk

import (
	"strings"
	"testing"

	"github.com/Laloww/Forage/internal/loader"
)

func TestSplit_SingleDoc(t *testing.T) {
	docs := []loader.Document{
		{Path: "test.md", Content: "Hello world"},
	}

	chunks := Split(docs, DefaultOptions())
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	if chunks[0].DocPath != "test.md" {
		t.Errorf("expected DocPath test.md, got %s", chunks[0].DocPath)
	}
	if chunks[0].Text != "Hello world" {
		t.Errorf("unexpected text: %q", chunks[0].Text)
	}
	if chunks[0].Index != 0 {
		t.Errorf("expected index 0, got %d", chunks[0].Index)
	}
}

func TestSplit_MultipleParagraphs(t *testing.T) {
	content := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
	docs := []loader.Document{
		{Path: "doc.md", Content: content},
	}

	chunks := Split(docs, DefaultOptions())
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}

	// With default 512 size, all three paragraphs should fit in one chunk
	combined := chunks[0].Text
	if !strings.Contains(combined, "First") || !strings.Contains(combined, "Third") {
		t.Errorf("expected all paragraphs in one chunk, got: %q", combined)
	}
}

func TestSplit_LargeDocCreatesMultipleChunks(t *testing.T) {
	// Create content larger than chunk size
	var paragraphs []string
	for i := 0; i < 20; i++ {
		paragraphs = append(paragraphs, strings.Repeat("word ", 30))
	}
	content := strings.Join(paragraphs, "\n\n")

	docs := []loader.Document{
		{Path: "big.md", Content: content},
	}

	opts := Options{Size: 200, Overlap: 32}
	chunks := Split(docs, opts)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for large doc, got %d", len(chunks))
	}

	for i, c := range chunks {
		if c.DocPath != "big.md" {
			t.Errorf("chunk %d: expected DocPath big.md, got %s", i, c.DocPath)
		}
		if c.ID == "" {
			t.Errorf("chunk %d: empty ID", i)
		}
	}
}

func TestSplit_EmptyDoc(t *testing.T) {
	docs := []loader.Document{
		{Path: "empty.md", Content: ""},
	}

	chunks := Split(docs, DefaultOptions())
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty doc, got %d", len(chunks))
	}
}

func TestSplit_MultipleDocs(t *testing.T) {
	docs := []loader.Document{
		{Path: "a.md", Content: "Alpha content"},
		{Path: "b.md", Content: "Beta content"},
	}

	chunks := Split(docs, DefaultOptions())
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	paths := map[string]bool{}
	for _, c := range chunks {
		paths[c.DocPath] = true
	}
	if !paths["a.md"] || !paths["b.md"] {
		t.Error("expected chunks from both docs")
	}
}

func TestSplit_ChunkIDsUnique(t *testing.T) {
	docs := []loader.Document{
		{Path: "doc.md", Content: strings.Repeat("paragraph text here\n\n", 20)},
	}

	chunks := Split(docs, Options{Size: 50, Overlap: 10})
	seen := map[string]bool{}
	for _, c := range chunks {
		if seen[c.ID] {
			t.Fatalf("duplicate chunk ID: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Size != 512 {
		t.Errorf("expected default size 512, got %d", opts.Size)
	}
	if opts.Overlap != 64 {
		t.Errorf("expected default overlap 64, got %d", opts.Overlap)
	}
}
