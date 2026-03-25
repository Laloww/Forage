package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nSome content here."), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := LoadDir(path)
	if err != nil {
		t.Fatalf("LoadDir single file: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Path != path {
		t.Errorf("expected path %s, got %s", path, docs[0].Path)
	}
	if docs[0].Content != "# Hello\n\nSome content here." {
		t.Errorf("unexpected content: %q", docs[0].Content)
	}
}

func TestLoadDir_Directory(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"readme.md":  "# Readme",
		"notes.txt":  "Some notes",
		"main.go":    "package main",
		"photo.png":  "not a text file",
		"secrets.env": "SECRET=abc",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	docs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}

	// .md, .txt, .go should load; .png unsupported; .env explicitly skipped
	if len(docs) != 3 {
		names := make([]string, len(docs))
		for i, d := range docs {
			names[i] = filepath.Base(d.Path)
		}
		t.Fatalf("expected 3 docs, got %d: %v", len(docs), names)
	}
}

func TestLoadDir_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	hiddenDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "config.txt"), []byte("git config"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("visible"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc (hidden dir skipped), got %d", len(docs))
	}
	if filepath.Base(docs[0].Path) != "visible.txt" {
		t.Errorf("expected visible.txt, got %s", filepath.Base(docs[0].Path))
	}
}

func TestLoadDir_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()

	nm := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "pkg.js"), []byte("module.exports={}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log('hi')"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc (node_modules skipped), got %d", len(docs))
	}
}

func TestLoadDir_EmptyFileSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "empty.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc (empty skipped), got %d", len(docs))
	}
}

func TestLoadDir_NonExistentPath(t *testing.T) {
	_, err := LoadDir("/nonexistent/path/12345")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestLoadDir_Recursive(t *testing.T) {
	dir := t.TempDir()

	sub := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "top.md"), []byte("top"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.md"), []byte("deep"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs (recursive), got %d", len(docs))
	}
}
