package chunk

import (
	"strings"

	"github.com/Laloww/Forage/internal/loader"
)

// Chunk represents a piece of a document with metadata.
type Chunk struct {
	ID      string
	DocPath string
	Text    string
	Index   int
}

// Options controls chunking behavior.
type Options struct {
	Size    int
	Overlap int
}

// DefaultOptions returns sensible defaults for chunking.
func DefaultOptions() Options {
	return Options{
		Size:    512,
		Overlap: 64,
	}
}

// Split breaks documents into overlapping text chunks.
func Split(docs []loader.Document, opts Options) []Chunk {
	var chunks []Chunk

	for _, doc := range docs {
		paragraphs := splitParagraphs(doc.Content)
		merged := mergeParagraphs(paragraphs, opts.Size)

		for i, text := range merged {
			c := Chunk{
				ID:      chunkID(doc.Path, i),
				DocPath: doc.Path,
				Text:    text,
				Index:   i,
			}
			chunks = append(chunks, c)
		}
	}

	return chunks
}

func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	var result []string
	for _, p := range raw {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func mergeParagraphs(paragraphs []string, maxSize int) []string {
	if len(paragraphs) == 0 {
		return nil
	}

	var result []string
	var current strings.Builder

	for _, p := range paragraphs {
		if current.Len() > 0 && current.Len()+len(p)+2 > maxSize {
			result = append(result, strings.TrimSpace(current.String()))
			// keep last paragraph for overlap
			current.Reset()
			if len(result) > 0 {
				lastChunk := result[len(result)-1]
				lines := strings.Split(lastChunk, "\n")
				if len(lines) > 1 {
					current.WriteString(lines[len(lines)-1])
					current.WriteString("\n\n")
				}
			}
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(p)
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

func chunkID(path string, index int) string {
	return strings.ReplaceAll(path, "/", "_") + "_" + itoa(index)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
