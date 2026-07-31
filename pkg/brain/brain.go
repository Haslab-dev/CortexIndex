package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const brainFile = "brain.md"

func Path(cortexDir string) string {
	return filepath.Join(cortexDir, brainFile)
}

func Init(cortexDir, projectName string) error {
	p := Path(cortexDir)
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	content := "# Brain\n"
	return os.WriteFile(p, []byte(content), 0644)
}

func Get(cortexDir string) (string, error) {
	p := Path(cortexDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("brain.md not found — run 'cortex init' first")
		}
		return "", err
	}
	return string(data), nil
}

func Update(cortexDir, content string) (string, error) {
	p := Path(cortexDir)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		return "", err
	}
	return content, nil
}

func Append(cortexDir, line string) (string, error) {
	existing, err := Get(cortexDir)
	if err != nil {
		existing = "# Brain\n"
	}
	timestamp := time.Now().Format("2006-01-02 15:04")
	newLine := fmt.Sprintf("[%s] %s", timestamp, line)
	updated := strings.TrimRight(existing, "\n") + "\n" + newLine + "\n"
	_, err = Update(cortexDir, updated)
	return newLine, err
}

func Search(cortexDir, query string) ([]string, error) {
	content, err := Get(cortexDir)
	if err != nil {
		return nil, err
	}
	var results []string
	q := strings.ToLower(query)
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(strings.ToLower(line), q) && !strings.HasPrefix(line, "#") {
			results = append(results, line)
		}
	}
	return results, nil
}

func Prune(cortexDir string) error {
	_, err := Update(cortexDir, "# Brain\n")
	return err
}
