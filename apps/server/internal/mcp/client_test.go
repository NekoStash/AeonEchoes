package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"aeonechoes/server/internal/domain"
)

func TestHTTPListTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Plain") != "plain" {
			t.Fatalf("expected plain header")
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("expected secret header")
		}

		var request JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.JSONRPC != jsonrpcVersion || request.ID != 1 || request.Method != "tools/list" {
			t.Fatalf("unexpected request: %+v", request)
		}

		writeJSON(t, w, JSONRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      1,
			Result:  json.RawMessage(`{"tools":[{"name":"read_file","description":"Read a file"}]}`),
		})
	}))
	defer server.Close()

	client := newTestClient(t, domain.MCPServerConfig{
		Name:      "http-test",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       server.URL,
		Headers: map[string]string{
			"X-Plain": "plain",
		},
		SecretHeaders: map[string]string{
			"Authorization": "Bearer secret",
		},
	})

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "read_file" || tools[0].Description != "Read a file" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestHTTPCallTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      int             `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if raw.Method != "tools/call" {
			t.Fatalf("unexpected method %q", raw.Method)
		}

		var params toolsCallParams
		if err := json.Unmarshal(raw.Params, &params); err != nil {
			t.Fatalf("decode params: %v", err)
		}
		if params.Name != "echo" {
			t.Fatalf("unexpected tool name %q", params.Name)
		}
		args, ok := params.Arguments.(map[string]any)
		if !ok || args["message"] != "hello" {
			t.Fatalf("unexpected arguments: %#v", params.Arguments)
		}

		writeJSON(t, w, JSONRPCResponse{
			JSONRPC: jsonrpcVersion,
			ID:      1,
			Result:  json.RawMessage(`{"content":[{"type":"text","text":"hello"}]}`),
		})
	}))
	defer server.Close()

	client := newTestClient(t, domain.MCPServerConfig{
		Name:      "http-call-test",
		Transport: domain.MCPTransportStreamableHTTP,
		URL:       server.URL,
	})

	result, err := client.CallTool(context.Background(), "echo", map[string]string{"message": "hello"})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "text" || result.Content[0].Text != "hello" {
		t.Fatalf("unexpected call result: %+v", result)
	}
}

func TestHTTPNon2xxFailsFast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad mcp", http.StatusBadGateway)
	}))
	defer server.Close()

	client := newTestClient(t, domain.MCPServerConfig{
		Name:      "http-fail-test",
		Transport: domain.MCPTransportSSE,
		URL:       server.URL,
	})

	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 502") || !strings.Contains(err.Error(), "bad mcp") {
		t.Fatalf("expected status and body in error, got %v", err)
	}
}

func TestStdioListToolsAndEnv(t *testing.T) {
	client := newTestClient(t, domain.MCPServerConfig{
		Name:      "stdio-list-test",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestHelperProcess", "--", "tools-list"},
		Env: map[string]string{
			"MCP_VISIBLE_ENV": "visible",
		},
		SecretEnv: map[string]string{
			"MCP_SECRET_ENV": "secret",
		},
	})

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "stdio_tool" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestStdioCallTool(t *testing.T) {
	client := newTestClient(t, domain.MCPServerConfig{
		Name:      "stdio-call-test",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestHelperProcess", "--", "tools-call"},
	})

	result, err := client.CallTool(context.Background(), "echo", map[string]string{"message": "hello"})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "stdio hello" {
		t.Fatalf("unexpected call result: %+v", result)
	}
}

func TestStdioErrorIncludesStderr(t *testing.T) {
	client := newTestClient(t, domain.MCPServerConfig{
		Name:      "stdio-error-test",
		Transport: domain.MCPTransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestHelperProcess", "--", "fail"},
	})

	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "helper stderr failure") {
		t.Fatalf("expected stderr in error, got %v", err)
	}
}

func TestNewClientValidatesTransport(t *testing.T) {
	_, err := NewClient(domain.MCPServerConfig{
		Name:      "bad-transport",
		Transport: domain.MCPTransport("websocket"),
	}, time.Second)
	if err == nil {
		t.Fatal("expected invalid transport error")
	}
}

func TestHelperProcess(t *testing.T) {
	args := os.Args
	separator := -1
	for index, arg := range args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator == -1 {
		return
	}

	mode := args[separator+1]
	switch mode {
	case "tools-list":
		var request JSONRPCRequest
		if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
			fmt.Fprintf(os.Stderr, "decode helper request: %v\n", err)
			os.Exit(2)
		}
		if request.Method != "tools/list" {
			fmt.Fprintf(os.Stderr, "unexpected method %s\n", request.Method)
			os.Exit(2)
		}
		if os.Getenv("MCP_VISIBLE_ENV") != "visible" || os.Getenv("MCP_SECRET_ENV") != "secret" {
			fmt.Fprintln(os.Stderr, "expected merged env")
			os.Exit(2)
		}
		fmt.Println(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"stdio_tool"}]}}`)
	case "tools-call":
		var raw struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(os.Stdin).Decode(&raw); err != nil {
			fmt.Fprintf(os.Stderr, "decode helper request: %v\n", err)
			os.Exit(2)
		}
		if raw.Method != "tools/call" {
			fmt.Fprintf(os.Stderr, "unexpected method %s\n", raw.Method)
			os.Exit(2)
		}
		fmt.Println(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"stdio hello"}]}}`)
	case "fail":
		fmt.Fprintln(os.Stderr, "helper stderr failure")
		os.Exit(3)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %s\n", mode)
		os.Exit(2)
	}
	os.Exit(0)
}

func newTestClient(t *testing.T, cfg domain.MCPServerConfig) *Client {
	t.Helper()
	client, err := NewClient(cfg, time.Second)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
