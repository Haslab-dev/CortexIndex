package mcp

import (
	"context"
	"fmt"

	"github.com/cortex-index/cortex/pkg/brain"
	"github.com/cortex-index/cortex/pkg/knowledge"
	"github.com/cortex-index/cortex/pkg/search"
	"github.com/cortex-index/cortex/pkg/skills"
	"github.com/cortex-index/cortex/pkg/tasks"
)

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func (s *Server) registerDefaultTools() {
	// 1. cortex_search
	s.tools["cortex_search"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query parameter is required")
		}
		return search.Search(s.workspacePath, s.cortexDir, query)
	}

	// 2. cortex_grep
	s.tools["cortex_grep"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query parameter is required")
		}
		return search.Grep(s.workspacePath, query)
	}

	// 3. cortex_glob
	s.tools["cortex_glob"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		pattern, _ := args["pattern"].(string)
		if pattern == "" {
			return nil, fmt.Errorf("pattern parameter is required")
		}
		return search.Glob(s.workspacePath, pattern)
	}

	// 4. cortex_brain_get — read brain.md
	s.tools["cortex_brain_get"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		if s.cortexDir == "" {
			return nil, fmt.Errorf("cortex directory not set")
		}
		content, err := brain.Get(s.cortexDir)
		if err != nil {
			return nil, err
		}
		return content, nil
	}

	// 5. cortex_brain_update — update brain.md
	s.tools["cortex_brain_update"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		if s.cortexDir == "" {
			return nil, fmt.Errorf("cortex directory not set")
		}
		content, _ := args["content"].(string)
		if content == "" {
			return nil, fmt.Errorf("content parameter is required")
		}
		result, err := brain.Update(s.cortexDir, content)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"status":  "Learned",
			"content": result,
		}, nil
	}

	// 6. cortex_brain_append — append to brain.md
	s.tools["cortex_brain_append"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		if s.cortexDir == "" {
			return nil, fmt.Errorf("cortex directory not set")
		}
		line, _ := args["line"].(string)
		if line == "" {
			return nil, fmt.Errorf("line parameter is required")
		}
		result, err := brain.Append(s.cortexDir, line)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"status":  "Learned",
			"content": result,
		}, nil
	}

	// 7. cortex_skill_install
	s.tools["cortex_skill_install"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		global, _ := args["global"].(bool)
		content, _ := args["content"].(string)
		res, err := skills.Install(s.cortexDir, name, global, content)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 8. cortex_skill_uninstall
	s.tools["cortex_skill_uninstall"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		global, _ := args["global"].(bool)
		err := skills.Uninstall(s.cortexDir, name, global)
		if err != nil {
			return nil, err
		}
		return "skill uninstalled", nil
	}

	// 9. cortex_skill_list
	s.tools["cortex_skill_list"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		res, err := skills.List(s.cortexDir)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 10. cortex_skill_search
	s.tools["cortex_skill_search"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		res, err := skills.Search(s.cortexDir, query)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 11. cortex_skill_update
	s.tools["cortex_skill_update"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		global, _ := args["global"].(bool)
		content, _ := args["content"].(string)
		res, err := skills.Install(s.cortexDir, name, global, content)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 12. cortex_task_create
	s.tools["cortex_task_create"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		goal, _ := args["goal"].(string)
		reqs, _ := args["requirements"].(string)
		if name == "" || goal == "" {
			return nil, fmt.Errorf("name and goal are required")
		}
		res, err := tasks.Create(s.cortexDir, name, goal, reqs)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 13. cortex_task_update
	s.tools["cortex_task_update"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		status, _ := args["status"].(string)
		if name == "" || status == "" {
			return nil, fmt.Errorf("name and status are required")
		}
		err := tasks.UpdateStatus(s.cortexDir, name, status)
		if err != nil {
			return nil, err
		}
		return "task updated", nil
	}

	// 14. cortex_task_complete
	s.tools["cortex_task_complete"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		err := tasks.Complete(s.cortexDir, name)
		if err != nil {
			return nil, err
		}
		return "task completed", nil
	}

	// 15. cortex_task_list
	s.tools["cortex_task_list"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		res, err := tasks.List(s.cortexDir)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 16. cortex_task_show
	s.tools["cortex_task_show"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		res, err := tasks.Show(s.cortexDir, name)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 17. cortex_task_delete
	s.tools["cortex_task_delete"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		err := tasks.Delete(s.cortexDir, name)
		if err != nil {
			return nil, err
		}
		return "task deleted", nil
	}

	// 18. cortex_codebase_init
	s.tools["cortex_codebase_init"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		err := knowledge.InitCodebase(s.cortexDir)
		if err != nil {
			return nil, err
		}
		return "codebase initialized", nil
	}

	// 19. cortex_codebase_update
	s.tools["cortex_codebase_update"] = func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		err := knowledge.UpdateCodebase(s.cortexDir, s.workspacePath)
		if err != nil {
			return nil, err
		}
		return "codebase updated", nil
	}
}

func emptySchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func stringParamSchema(paramName, description string, required bool) map[string]interface{} {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			paramName: map[string]interface{}{
				"type":        "string",
				"description": description,
			},
		},
	}
	if required {
		schema["required"] = []string{paramName}
	}
	return schema
}

func (s *Server) listToolDefinitions() []toolDef {
	return []toolDef{
		{Name: "cortex_search", Description: "Search knowledge base files and general codebase files recursively", InputSchema: stringParamSchema("query", "Search query", true)},
		{Name: "cortex_grep", Description: "Search for target text within workspace files", InputSchema: stringParamSchema("query", "Search substring query", true)},
		{Name: "cortex_glob", Description: "Match file paths in workspace using glob pattern", InputSchema: stringParamSchema("pattern", "Glob pattern (e.g. *.go)", true)},
		{Name: "cortex_brain_get", Description: "Read the project's brain.md (purpose, rules, permissions, history)", InputSchema: emptySchema()},
		{Name: "cortex_brain_update", Description: "Replace the entire brain.md with new content", InputSchema: stringParamSchema("content", "Full brain.md content", true)},
		{Name: "cortex_brain_append", Description: "Append a line of knowledge/policy to brain.md (marked Learned)", InputSchema: stringParamSchema("line", "Knowledge line/policy to append", true)},
		{Name: "cortex_skill_install", Description: "Install a coding skill conventions template (workspace/global)", InputSchema: stringParamSchema("name", "Skill name", true)},
		{Name: "cortex_skill_uninstall", Description: "Uninstall a coding skill conventions template", InputSchema: stringParamSchema("name", "Skill name", true)},
		{Name: "cortex_skill_list", Description: "List all global and workspace skills", InputSchema: emptySchema()},
		{Name: "cortex_skill_search", Description: "Search for skills by query text", InputSchema: stringParamSchema("query", "Search query", true)},
		{Name: "cortex_skill_update", Description: "Update or install coding skill content", InputSchema: stringParamSchema("name", "Skill name", true)},
		{Name: "cortex_task_create", Description: "Create a task tracking document in .cortex/tasks/", InputSchema: stringParamSchema("name", "Task name", true)},
		{Name: "cortex_task_update", Description: "Update active task status state", InputSchema: stringParamSchema("name", "Task name", true)},
		{Name: "cortex_task_complete", Description: "Mark a task tracking document completed", InputSchema: stringParamSchema("name", "Task name", true)},
		{Name: "cortex_task_list", Description: "List all registered workspace tasks", InputSchema: emptySchema()},
		{Name: "cortex_task_show", Description: "Show detailed task instructions", InputSchema: stringParamSchema("name", "Task name", true)},
		{Name: "cortex_task_delete", Description: "Delete/remove task document from workspace", InputSchema: stringParamSchema("name", "Task name", true)},
		{Name: "cortex_codebase_init", Description: "Initialize codebase.md architecture overview file", InputSchema: emptySchema()},
		{Name: "cortex_codebase_update", Description: "Update codebase.md metadata recursively without database dependencies", InputSchema: emptySchema()},
	}
}
