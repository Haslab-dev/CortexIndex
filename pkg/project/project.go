package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cortexDirName = ".cortex"

// ProjectMetadata stored in .cortex/project.json (PRD 2.0 Schema)
type ProjectMetadata struct {
	Version   string `json:"version"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Language  string `json:"language"`
	Framework string `json:"framework"`
	Generator string `json:"generator"`
}

// ProjectContext is the central context every subsystem receives.
type ProjectContext struct {
	Root      string
	CortexDir string
	Meta      ProjectMetadata
}

// DetectProject auto-detects language and framework based on project files
func DetectProject(root string) (string, string) {
	lang := "Unknown"
	fw := "None"

	// 1. Check Go
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return "Go", "None"
	}

	// 2. Check Rust
	if _, err := os.Stat(filepath.Join(root, "Cargo.toml")); err == nil {
		return "Rust", "None"
	}

	// 3. Check Python
	if _, err := os.Stat(filepath.Join(root, "requirements.txt")); err == nil {
		return "Python", "None"
	}
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
		return "Python", "None"
	}

	// 4. Check JS/TS
	pkgJSONPath := filepath.Join(root, "package.json")
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		lang = "JavaScript"
		if _, err := os.Stat(filepath.Join(root, "tsconfig.json")); err == nil {
			lang = "TypeScript"
		}

		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			hasDep := func(name string) bool {
				_, ok1 := pkg.Dependencies[name]
				_, ok2 := pkg.DevDependencies[name]
				return ok1 || ok2
			}
			if hasDep("next") {
				fw = "Next.js"
			} else if hasDep("react") {
				fw = "React"
			} else if hasDep("vue") {
				fw = "Vue"
			} else if hasDep("svelte") || hasDep("@sveltejs/kit") {
				fw = "Svelte"
			}
		}
	}
	return lang, fw
}

// Discover walks up from dir looking for .cortex/ directory with project.json.
func Discover(dir string) *ProjectContext {
	current := dir
	for i := 0; i < 20; i++ {
		cd := filepath.Join(current, cortexDirName)
		fi, err := os.Stat(cd)
		if err != nil || !fi.IsDir() {
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
			continue
		}
		meta, err := loadMeta(cd)
		if err != nil || meta.Version == "" {
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
			continue
		}
		return &ProjectContext{
			Root:      current,
			CortexDir: cd,
			Meta:      meta,
		}
	}
	return nil
}

// Init creates a .cortex/ directory in root with project.json.
func Init(root string) (*ProjectContext, error) {
	cd := filepath.Join(root, cortexDirName)
	if fi, err := os.Stat(cd); err == nil && fi.IsDir() {
		if _, err := loadMeta(cd); err == nil {
			return nil, fmt.Errorf(".cortex already exists in %s", root)
		}
	} else {
		if err := os.MkdirAll(cd, 0755); err != nil {
			return nil, fmt.Errorf("failed creating %s: %w", cd, err)
		}
	}

	lang, fw := DetectProject(root)
	nowStr := time.Now().Format(time.RFC3339)
	meta := ProjectMetadata{
		Version:   "2.0",
		CreatedAt: nowStr,
		UpdatedAt: nowStr,
		Language:  lang,
		Framework: fw,
		Generator: "Cortex",
	}
	if err := saveMeta(cd, meta); err != nil {
		return nil, err
	}

	// Register .cortex to .gitignore if it exists
	gitignorePath := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		if content, err := os.ReadFile(gitignorePath); err == nil {
			strContent := string(content)
			lines := strings.Split(strContent, "\n")
			hasCortex := false
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == ".cortex" || trimmed == ".cortex/" {
					hasCortex = true
					break
				}
			}
			if !hasCortex {
				if len(strContent) > 0 && !strings.HasSuffix(strContent, "\n") {
					strContent += "\n"
				}
				strContent += ".cortex\n"
				_ = os.WriteFile(gitignorePath, []byte(strContent), 0644)
			}
		}
	}

	return &ProjectContext{
		Root:      root,
		CortexDir: cd,
		Meta:      meta,
	}, nil
}

func (pc *ProjectContext) Update() error {
	lang, fw := DetectProject(pc.Root)
	pc.Meta.Language = lang
	pc.Meta.Framework = fw
	pc.Meta.UpdatedAt = time.Now().Format(time.RFC3339)
	return saveMeta(pc.CortexDir, pc.Meta)
}

func loadMeta(cortexDir string) (ProjectMetadata, error) {
	var meta ProjectMetadata
	data, err := os.ReadFile(filepath.Join(cortexDir, "project.json"))
	if err != nil {
		return meta, err
	}
	_ = json.Unmarshal(data, &meta)
	return meta, nil
}

func saveMeta(cortexDir string, meta ProjectMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cortexDir, "project.json"), data, 0644)
}
