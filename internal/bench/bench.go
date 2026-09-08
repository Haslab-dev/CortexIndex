// Package bench provides a deterministic local comparison between bounded
// manual lexical exploration and Cortex retrieval. Token values are chars/4
// estimates, not provider billing or real LLM telemetry.
package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Row struct {
	Arm             string `json:"arm"`
	Iteration       int    `json:"iteration"`
	PrepDurationMS  int64  `json:"prep_duration_ms"`
	DurationMS      int64  `json:"duration_ms"`
	ToolCalls       int    `json:"tool_calls"`
	FilesRead       int    `json:"files_read"`
	LinesRead       int    `json:"lines_read"`
	InputChars      int    `json:"input_chars"`
	InputTokensEst  int    `json:"input_tokens_est"`
	OutputChars     int    `json:"output_chars"`
	OutputTokensEst int    `json:"output_tokens_est"`
	IndexedFiles    int    `json:"indexed_files,omitempty"`
	IndexedSymbols  int    `json:"indexed_symbols,omitempty"`
	Success         bool   `json:"success"`
	Failure         string `json:"failure,omitempty"`
}

type Report struct {
	Task        string `json:"task"`
	Iterations  int    `json:"iterations"`
	Methodology string `json:"methodology"`
	Rows        []Row  `json:"rows"`
}

func Run(executable string, iterations int) (Report, error) {
	if iterations < 1 {
		iterations = 1
	}
	root, err := os.MkdirTemp("", "cortex-bench-")
	if err != nil {
		return Report{}, err
	}
	defer os.RemoveAll(root)
	writeFixture(root)
	task := "How does authentication work and where is the login entry point?"
	report := Report{Task: task, Iterations: iterations, Methodology: "Deterministic local proxy: baseline scans bounded fixture files; Cortex runs its local index/context commands. Token values are len(chars)/4 estimates, not provider telemetry. Preparation/indexing time is separate from task latency."}
	for i := 1; i <= iterations; i++ {
		row, err := runBaseline(root, task, i)
		if err != nil {
			return report, err
		}
		report.Rows = append(report.Rows, row)
		prepStart := time.Now()
		prep, err := runCommand(executable, "index", "--dir", root)
		if err != nil {
			return report, err
		}
		prepMS := time.Since(prepStart).Milliseconds()
		row, err = runCortex(executable, root, task, i, prepMS, prep)
		if err != nil {
			return report, err
		}
		report.Rows = append(report.Rows, row)
	}
	return report, nil
}

func runBaseline(root, task string, iteration int) (Row, error) {
	start := time.Now()
	row := Row{Arm: "baseline", Iteration: iteration, InputChars: len(task), ToolCalls: 2}
	var output strings.Builder
	terms := []string{"auth", "login", "token"}
	files, err := textFiles(root)
	if err != nil {
		return row, err
	}
	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		row.FilesRead++
		lines := strings.Split(string(data), "\n")
		row.LinesRead += len(lines)
		low := strings.ToLower(string(data))
		for _, term := range terms {
			if strings.Contains(low, term) {
				output.WriteString(filepath.Base(p) + " ")
				break
			}
		}
	}
	row.OutputChars = len(output.String())
	row.OutputTokensEst = row.OutputChars / 4
	row.InputTokensEst = row.InputChars / 4
	row.DurationMS = time.Since(start).Milliseconds()
	row.Success = strings.Contains(strings.ToLower(output.String()), "server") || row.FilesRead > 0
	if !row.Success {
		row.Failure = "baseline found no relevant fixture files"
	}
	return row, nil
}

func runCortex(exe, root, task string, iteration int, prepMS int64, prep string) (Row, error) {
	start := time.Now()
	out, err := runCommand(exe, "context", task, "--dir", root)
	if err != nil {
		return Row{}, err
	}
	row := Row{Arm: "cortex", Iteration: iteration, PrepDurationMS: prepMS, DurationMS: time.Since(start).Milliseconds(), ToolCalls: 1, InputChars: len(task), InputTokensEst: len(task) / 4, OutputChars: len(out), OutputTokensEst: len(out) / 4}
	row.FilesRead = countBacktickPaths(out)
	row.LinesRead = countLines(out)
	row.Success = strings.Contains(out, "Server.Login") && strings.Contains(out, "Project Memory")
	if !row.Success {
		row.Failure = "context output missed expected symbol or memory"
	}
	if strings.Contains(prep, "Total files in index:") {
		row.IndexedFiles = parseMetric(prep, "Total files in index:")
		row.IndexedSymbols = parseMetric(prep, "Total symbols:")
	}
	return row, nil
}

func runCommand(exe string, args ...string) (string, error) {
	cmd := exec.Command(exe, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return string(data), fmt.Errorf("%s %v: %w", exe, args, err)
	}
	return string(data), nil
}
func writeFixture(root string) {
	files := map[string]string{".cortex/project.md": "# Project\n\nAuthentication service fixture.\n", "auth/server.go": "package auth\n\n// Server handles authentication.\ntype Server struct{}\n\nfunc (s *Server) Login(user string) error { return s.store.Issue(user) }\n\ntype TokenStore struct{}\nfunc (t *TokenStore) Issue(user string) error { return nil }\n", "main.go": "package main\nfunc main() {}\n", "docs.md": "Authentication and token flow notes.\n"}
	for rel, data := range files {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(data), 0644)
	}
}
func textFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, e error) error {
		if e != nil || info.IsDir() || strings.HasPrefix(filepath.ToSlash(p), filepath.ToSlash(filepath.Join(root, ".cortex", "index"))) {
			return nil
		}
		if filepath.Ext(p) == ".go" || filepath.Ext(p) == ".md" {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}
func countBacktickPaths(s string) int { return strings.Count(s, "`") / 2 }
func countLines(s string) int         { return len(strings.Split(s, "\n")) }
func parseMetric(s, key string) int {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, key) {
			var n int
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "+key)), "%d", &n)
			return n
		}
	}
	return 0
}
func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
func (r Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# Cortex Local Benchmark\n\n")
	b.WriteString("> Token counts are local chars/4 estimates, not provider billing.\n\n")
	b.WriteString(fmt.Sprintf("- Task: %s\n- Iterations: %d\n\n", r.Task, r.Iterations))
	b.WriteString("| Arm | Run ms | Prep ms | Calls | Files | Lines | Input est. | Output est. | Success |\n|---|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, x := range r.Rows {
		b.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %d | %d | %t |\n", x.Arm, x.DurationMS, x.PrepDurationMS, x.ToolCalls, x.FilesRead, x.LinesRead, x.InputTokensEst, x.OutputTokensEst, x.Success))
	}
	return b.String()
}
