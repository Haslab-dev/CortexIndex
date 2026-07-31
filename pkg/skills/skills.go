package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Content     string
	IsGlobal    bool
	FilePath    string
}

func GlobalDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cortex", "skills")
}

func WorkspaceDir(cortexDir string) string {
	return filepath.Join(cortexDir, "skills")
}

func ParseFrontmatter(content string) (name, desc, body string) {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		inFM := true
		fmLines := []string{}
		bodyLines := []string{}
		for i := 1; i < len(lines); i++ {
			line := lines[i]
			if inFM {
				if strings.TrimSpace(line) == "---" {
					inFM = false
					continue
				}
				fmLines = append(fmLines, line)
			} else {
				bodyLines = append(bodyLines, line)
			}
		}
		for _, line := range fmLines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if k == "name" {
					name = v
				} else if k == "description" {
					desc = v
				}
			}
		}
		body = strings.Join(bodyLines, "\n")
		return
	}
	body = content
	return
}

func Install(cortexDir, name string, global bool, content string) (*Skill, error) {
	var dir string
	if global {
		dir = GlobalDir()
	} else {
		dir = WorkspaceDir(cortexDir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, name+".md")
	if content == "" {
		content = fmt.Sprintf("---\nname: %s\ndescription: Auto-installed %s coding conventions\n---\n\nAlways follow best practices for %s.\n", name, name, name)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	parsedName, desc, body := ParseFrontmatter(content)
	if parsedName == "" {
		parsedName = name
	}

	return &Skill{
		Name:        parsedName,
		Description: desc,
		Content:     body,
		IsGlobal:    global,
		FilePath:    filePath,
	}, nil
}

func Uninstall(cortexDir, name string, global bool) error {
	var dir string
	if global {
		dir = GlobalDir()
	} else {
		dir = WorkspaceDir(cortexDir)
	}
	filePath := filepath.Join(dir, name+".md")
	return os.Remove(filePath)
}

func List(cortexDir string) ([]Skill, error) {
	var skills []Skill
	gDir := GlobalDir()
	_ = filepath.Walk(gDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err == nil {
				name := strings.TrimSuffix(info.Name(), ".md")
				parsedName, desc, body := ParseFrontmatter(string(content))
				if parsedName == "" {
					parsedName = name
				}
				skills = append(skills, Skill{
					Name:        parsedName,
					Description: desc,
					Content:     body,
					IsGlobal:    true,
					FilePath:    path,
				})
			}
		}
		return nil
	})

	wDir := WorkspaceDir(cortexDir)
	_ = filepath.Walk(wDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err == nil {
				name := strings.TrimSuffix(info.Name(), ".md")
				parsedName, desc, body := ParseFrontmatter(string(content))
				if parsedName == "" {
					parsedName = name
				}
				skills = append(skills, Skill{
					Name:        parsedName,
					Description: desc,
					Content:     body,
					IsGlobal:    false,
					FilePath:    path,
				})
			}
		}
		return nil
	})

	return skills, nil
}

func Search(cortexDir, query string) ([]Skill, error) {
	all, err := List(cortexDir)
	if err != nil {
		return nil, err
	}
	var results []Skill
	q := strings.ToLower(query)
	for _, s := range all {
		if strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.Description), q) ||
			strings.Contains(strings.ToLower(s.Content), q) {
			results = append(results, s)
		}
	}
	return results, nil
}
