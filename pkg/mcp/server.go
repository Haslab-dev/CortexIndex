package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type Server struct {
	cortexDir     string
	workspacePath string
	tools         map[string]ToolHandler
	mu            sync.RWMutex
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

func NewServer(cortexDir, workspacePath string) *Server {
	s := &Server{
		cortexDir:     cortexDir,
		workspacePath: workspacePath,
		tools:         make(map[string]ToolHandler),
	}
	s.registerDefaultTools()
	return s
}

func (s *Server) ServeStdio(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		os.Stdin.Close()
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	writer := os.Stdout

	if !scanner.Scan() {
		return nil
	}

	processLine := func(line string) {
		if len(line) == 0 {
			return
		}
		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			return
		}
		if req.ID == nil {
			_ = s.HandleRequest(ctx, req)
			return
		}
		resp := s.HandleRequest(ctx, req)
		respBytes, _ := json.Marshal(resp)
		respBytes = append(respBytes, '\n')
		_, _ = writer.Write(respBytes)
	}

	processLine(scanner.Text())

	for scanner.Scan() {
		processLine(scanner.Text())
	}

	return nil
}

func (s *Server) HandleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "cortex-index",
					"version": "2.0.0",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": true,
					},
				},
			},
		}

	case "notifications/initialized", "initialized":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	case "ping":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	case "tools/list":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.listToolDefinitions(),
			},
		}

	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}

		if params.Name == "" {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params: tool name required",
				},
			}
		}

		s.mu.RLock()
		handler, exists := s.tools[params.Name]
		s.mu.RUnlock()

		if !exists {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: fmt.Sprintf("Tool not found: %s", params.Name),
				},
			}
		}

		if params.Arguments == nil {
			params.Arguments = make(map[string]interface{})
		}

		result, err := handler(ctx, params.Arguments)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": err.Error(),
						},
					},
					"isError": true,
				},
			}
		}

		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": mustJSON(result),
					},
				},
			},
		}

	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request json", http.StatusBadRequest)
		return
	}

	resp := s.HandleRequest(r.Context(), req)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func mustJSON(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
