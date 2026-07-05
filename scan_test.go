package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirectories_Paths(t *testing.T) {
	// Create temp directory with some subdirs
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "project-a"), 0o755)
	os.MkdirAll(filepath.Join(root, "project-b"), 0o755)
	os.WriteFile(filepath.Join(root, "not-a-dir.txt"), []byte("hi"), 0o644)

	cfg := Config{Paths: []string{root}}
	entries, warnings := scanDirectories(cfg)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "project-a" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "project-a")
	}
	if entries[1].Name != "project-b" {
		t.Errorf("entries[1].Name = %q, want %q", entries[1].Name, "project-b")
	}
	if entries[0].ParentPath != root {
		t.Errorf("entries[0].ParentPath = %q, want %q", entries[0].ParentPath, root)
	}
}

func TestScanDirectories_Projects(t *testing.T) {
	// Create temp directories to use as individual projects
	projDir := t.TempDir()

	cfg := Config{Projects: []string{projDir}}
	entries, warnings := scanDirectories(cfg)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != filepath.Base(projDir) {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, filepath.Base(projDir))
	}
	if entries[0].Path != projDir {
		t.Errorf("entries[0].Path = %q, want %q", entries[0].Path, projDir)
	}
}

func TestScanDirectories_MissingPath(t *testing.T) {
	cfg := Config{Paths: []string{"/nonexistent/path/xyz"}}
	entries, warnings := scanDirectories(cfg)

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0] != "path not found: /nonexistent/path/xyz" {
		t.Errorf("unexpected warning: %q", warnings[0])
	}
}

func TestScanDirectories_MissingProject(t *testing.T) {
	cfg := Config{Projects: []string{"/nonexistent/project/xyz"}}
	entries, warnings := scanDirectories(cfg)

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0] != "project not found: /nonexistent/project/xyz" {
		t.Errorf("unexpected warning: %q", warnings[0])
	}
}

func TestScanDirectories_IncludesHidden(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hidden-project"), 0o755)
	os.MkdirAll(filepath.Join(root, "visible-project"), 0o755)

	cfg := Config{Paths: []string{root}}
	entries, warnings := scanDirectories(cfg)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (including hidden), got %d", len(entries))
	}
}

func TestScanDirectories_SortedByName(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "zulu"), 0o755)
	os.MkdirAll(filepath.Join(root, "alpha"), 0o755)
	os.MkdirAll(filepath.Join(root, "mike"), 0o755)

	cfg := Config{Paths: []string{root}}
	entries, _ := scanDirectories(cfg)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Name != "alpha" || entries[1].Name != "mike" || entries[2].Name != "zulu" {
		t.Errorf("entries not sorted: %v, %v, %v", entries[0].Name, entries[1].Name, entries[2].Name)
	}
}
