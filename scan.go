package main

import (
	"os"
	"path/filepath"
	"sort"
)

// DirEntry represents a single directory found under a configured scan path.
type DirEntry struct {
	Name       string // directory name (basename)
	Path       string // full absolute path
	ParentPath string // the configured scan path it came from (for display)
}

// scanDirectories reads all immediate subdirectories from each configured path,
// and includes individual project directories directly. Missing or unreadable
// paths are collected as warnings. Results are sorted alphabetically by name.
func scanDirectories(cfg Config) (entries []DirEntry, warnings []string) {
	// Scan children of each path
	for _, raw := range cfg.Paths {
		dir := expandPath(raw)

		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			warnings = append(warnings, "path not found: "+raw)
			continue
		}

		children, err := os.ReadDir(dir)
		if err != nil {
			warnings = append(warnings, "cannot read: "+raw)
			continue
		}

		for _, child := range children {
			if !child.IsDir() {
				continue
			}
			entries = append(entries, DirEntry{
				Name:       child.Name(),
				Path:       filepath.Join(dir, child.Name()),
				ParentPath: raw,
			})
		}
	}

	// Add individual project directories
	for _, raw := range cfg.Projects {
		dir := expandPath(raw)

		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			warnings = append(warnings, "project not found: "+raw)
			continue
		}

		entries = append(entries, DirEntry{
			Name:       filepath.Base(dir),
			Path:       dir,
			ParentPath: raw,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries, warnings
}
