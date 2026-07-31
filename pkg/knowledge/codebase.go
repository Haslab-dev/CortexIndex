package knowledge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CodebasePath(cortexDir string) string {
	return filepath.Join(cortexDir, "codebase.md")
}

func InitCodebase(cortexDir string) error {
	p := CodebasePath(cortexDir)
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	content := `# Overview

# Architecture

# Modules

# Dependencies

# Entrypoints

# Conventions

# Libraries
`
	return os.WriteFile(p, []byte(content), 0644)
}

func GetCodebase(cortexDir string) (string, error) {
	p := CodebasePath(cortexDir)
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func UpdateCodebase(cortexDir string, projectRoot string) error {
	p := CodebasePath(cortexDir)

	// Default values
	overview := ""
	architecture := ""
	dependencies := ""
	conventions := ""
	libraries := ""

	// Read existing codebase.md if it exists to preserve user-edited sections
	if data, err := os.ReadFile(p); err == nil {
		content := string(data)
		sections := splitSections(content)
		if v, ok := sections["Overview"]; ok {
			overview = v
		}
		if v, ok := sections["Architecture"]; ok {
			architecture = v
		}
		if v, ok := sections["Dependencies"]; ok {
			dependencies = v
		}
		if v, ok := sections["Conventions"]; ok {
			conventions = v
		}
		if v, ok := sections["Libraries"]; ok {
			libraries = v
		}
	}

	// Scan project Root directory
	modules := make(map[string]int) // dirPath -> fileCount
	var entrypoints []string

	_ = filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip common ignored folders
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == ".cortex" || name == "node_modules" || name == "dist" || name == "bin" || name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// It is a file
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}

		// Count files by directory to represent modules
		dir := filepath.Dir(rel)
		if dir != "." {
			modules[dir]++
		}

		// Identify common entrypoints
		base := filepath.Base(rel)
		low := strings.ToLower(base)
		if low == "main.go" || low == "index.ts" || low == "app.tsx" || low == "main.ts" || low == "main.py" || low == "app.py" || low == "server.ts" || low == "server.js" || low == "index.js" {
			entrypoints = append(entrypoints, rel)
		}

		return nil
	})

	// Format modules section
	var modLines []string
	if len(modules) == 0 {
		modLines = append(modLines, "No subdirectories with files found.")
	} else {
		for dir, count := range modules {
			modLines = append(modLines, fmt.Sprintf("- **%s**: %d files", dir, count))
		}
	}

	// Format entrypoints section
	var entryLines []string
	if len(entrypoints) == 0 {
		entryLines = append(entryLines, "No common entrypoints detected.")
	} else {
		for _, ep := range entrypoints {
			entryLines = append(entryLines, fmt.Sprintf("- **%s**", ep))
		}
	}

	// Assemble final content
	var out strings.Builder
	out.WriteString("# Overview\n")
	if overview != "" {
		out.WriteString(overview + "\n\n")
	} else {
		out.WriteString("This codebase is managed with Cortex 2.0.\n\n")
	}

	out.WriteString("# Architecture\n")
	if architecture != "" {
		out.WriteString(architecture + "\n\n")
	} else {
		out.WriteString("Incremental module-based design.\n\n")
	}

	out.WriteString("# Modules\n")
	out.WriteString(strings.Join(modLines, "\n") + "\n\n")

	out.WriteString("# Dependencies\n")
	if dependencies != "" {
		out.WriteString(dependencies + "\n\n")
	} else {
		out.WriteString("Resolved dependencies via workspace structure.\n\n")
	}

	out.WriteString("# Entrypoints\n")
	out.WriteString(strings.Join(entryLines, "\n") + "\n\n")

	out.WriteString("# Conventions\n")
	if conventions != "" {
		out.WriteString(conventions + "\n\n")
	} else {
		out.WriteString("Project-specific coding guidelines.\n\n")
	}

	out.WriteString("# Libraries\n")
	if libraries != "" {
		out.WriteString(libraries + "\n")
	} else {
		out.WriteString("Third-party dependencies and frameworks.\n")
	}

	return os.WriteFile(p, []byte(out.String()), 0644)
}

func splitSections(content string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(content, "\n")
	var currentSection string
	var currentLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(strings.Join(currentLines, "\n"))
			}
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			currentLines = nil
		} else {
			if currentSection != "" {
				currentLines = append(currentLines, line)
			}
		}
	}
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(strings.Join(currentLines, "\n"))
	}
	return sections
}
