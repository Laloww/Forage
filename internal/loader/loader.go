package loader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Document represents a loaded file with its content and metadata.
type Document struct {
	Path    string
	Content string
}

// SupportedExtensions defines which file types can be indexed.
var SupportedExtensions = map[string]bool{
	".md":   true,
	".txt":  true,
	".go":   true,
	".py":   true,
	".js":   true,
	".ts":   true,
	".rs":   true,
	".yaml": true,
	".yml":  true,
	".json": true,
	".toml": true,
	".cfg":  true,
	".ini":  true,
	".sh":   true,
	".html": true,
	".css":  true,
	".xml":  true,
	".csv":  true,
	".log":  true,
	".env":  false, // explicitly skip secrets
}

// LoadDir recursively loads all supported files from a directory.
func LoadDir(root string) ([]Document, error) {
	var docs []Document

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", root, err)
	}

	if !info.IsDir() {
		doc, err := loadFile(root)
		if err != nil {
			return nil, err
		}
		return []Document{doc}, nil
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		supported, exists := SupportedExtensions[ext]
		if !exists || !supported {
			return nil
		}

		doc, err := loadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			return nil
		}
		docs = append(docs, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", root, err)
	}

	return docs, nil
}

func loadFile(path string) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("reading %s: %w", path, err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return Document{}, fmt.Errorf("empty file: %s", path)
	}

	return Document{
		Path:    path,
		Content: content,
	}, nil
}
