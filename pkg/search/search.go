package search

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Grep searches for target text within the project files, excluding hidden folders.
func Grep(root, query string) ([]string, error) {
	var matches []string
	q := strings.ToLower(query)

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".cortex" || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		// Simple text file check (skip binaries)
		ext := filepath.Ext(path)
		if isBinaryExt(ext) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		rel, _ := filepath.Rel(root, path)
		for lineNum, line := range lines {
			if strings.Contains(strings.ToLower(line), q) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", rel, lineNum+1, strings.TrimSpace(line)))
				if len(matches) >= 100 {
					return filepath.SkipAll // Limit results
				}
			}
		}
		return nil
	})

	return matches, err
}

// Glob returns all files matching pattern in the project
func Glob(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".cortex" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		matched, err := filepath.Match(pattern, info.Name())
		if err == nil && matched {
			matches = append(matches, rel)
		} else {
			matchedRel, errRel := filepath.Match(pattern, rel)
			if errRel == nil && matchedRel {
				matches = append(matches, rel)
			}
		}
		return nil
	})
	return matches, err
}

// Search queries both knowledge files and general codebase files
func Search(root, cortexDir, query string) ([]string, error) {
	var results []string
	q := strings.ToLower(query)

	// 1. Search Knowledge files first
	knowledgeFiles := []string{
		filepath.Join(cortexDir, "codebase.md"),
		filepath.Join(cortexDir, "brain.md"),
	}

	// Add task files
	tasksDir := filepath.Join(cortexDir, "tasks")
	if entries, err := os.ReadDir(tasksDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				knowledgeFiles = append(knowledgeFiles, filepath.Join(tasksDir, e.Name()))
			}
		}
	}

	// Add skill files
	skillsDir := filepath.Join(cortexDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				knowledgeFiles = append(knowledgeFiles, filepath.Join(skillsDir, e.Name()))
			}
		}
	}

	for _, kf := range knowledgeFiles {
		if data, err := os.ReadFile(kf); err == nil {
			rel, _ := filepath.Rel(root, kf)
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), q) {
					results = append(results, fmt.Sprintf("[Knowledge] %s:%d: %s", rel, i+1, strings.TrimSpace(line)))
				}
			}
		}
	}

	// 2. Grep remaining code files if we need more results
	if len(results) < 20 {
		codeMatches, _ := Grep(root, query)
		for _, cm := range codeMatches {
			results = append(results, "[Code] "+cm)
		}
	}

	return results, nil
}

func isBinaryExt(ext string) bool {
	binaryExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".zip":  true,
		".gz":   true,
		".tar":  true,
		".exe":  true,
		".bin":  true,
		".pdf":  true,
		".db":   true,
	}
	return binaryExts[strings.ToLower(ext)]
}
