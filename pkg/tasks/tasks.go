package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Task struct {
	Name         string
	Goal         string
	Status       string
	Requirements string
	Files        []string
	Symbols      []string
	Notes        string
	Progress     string
	FilePath     string
}

func Dir(cortexDir string) string {
	return filepath.Join(cortexDir, "tasks")
}

func Create(cortexDir, name, goal, reqs string) (*Task, error) {
	d := Dir(cortexDir)
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(d, name+".md")
	content := fmt.Sprintf(`# Task: %s

## Goal
%s

## Status
PLANNING

## Requirements
%s

## Files
- None

## Symbols
- None

## Notes
Created at %s

## Progress
- [ ] Task initialized
`, name, goal, reqs, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &Task{
		Name:         name,
		Goal:         goal,
		Status:       "PLANNING",
		Requirements: reqs,
		FilePath:     filePath,
	}, nil
}

func UpdateStatus(cortexDir, name, status string) error {
	filePath := filepath.Join(Dir(cortexDir), name+".md")
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	lines := strings.Split(content, "\n")
	foundStatus := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Status" {
			if i+1 < len(lines) {
				lines[i+1] = status
				foundStatus = true
				break
			}
		}
	}
	if !foundStatus {
		return fmt.Errorf("could not find status section in task file")
	}
	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}

func Complete(cortexDir, name string) error {
	filePath := filepath.Join(Dir(cortexDir), name+".md")
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	content = strings.Replace(content, "PLANNING", "COMPLETED", 1)
	content = strings.Replace(content, "IN_PROGRESS", "COMPLETED", 1)
	content = strings.Replace(content, "- [ ] Task initialized", "- [x] Task completed", 1)
	return os.WriteFile(filePath, []byte(content), 0644)
}

func Continue(cortexDir, name string) error {
	return UpdateStatus(cortexDir, name, "IN_PROGRESS")
}

func List(cortexDir string) ([]Task, error) {
	d := Dir(cortexDir)
	entries, err := os.ReadDir(d)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	tasksList := []Task{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			filePath := filepath.Join(d, e.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			content := string(data)
			goal := ""
			status := ""

			// Parse goal
			if goalIdx := strings.Index(content, "## Goal"); goalIdx != -1 {
				rest := content[goalIdx+7:]
				if nextSecIdx := strings.Index(rest, "##"); nextSecIdx != -1 {
					goal = strings.TrimSpace(rest[:nextSecIdx])
				}
			}
			// Parse status
			if statusIdx := strings.Index(content, "## Status"); statusIdx != -1 {
				rest := content[statusIdx+9:]
				if nextSecIdx := strings.Index(rest, "##"); nextSecIdx != -1 {
					status = strings.TrimSpace(rest[:nextSecIdx])
				}
			}
			tasksList = append(tasksList, Task{
				Name:     strings.TrimSuffix(e.Name(), ".md"),
				Goal:     goal,
				Status:   status,
				FilePath: filePath,
			})
		}
	}
	return tasksList, nil
}

func Show(cortexDir, name string) (string, error) {
	filePath := filepath.Join(Dir(cortexDir), name+".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Delete(cortexDir, name string) error {
	filePath := filepath.Join(Dir(cortexDir), name+".md")
	return os.Remove(filePath)
}
