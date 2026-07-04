package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
)

const jsonrpcVersion = "2.0"

// Config is the normalized MCP client configuration derived from the domain model.
type Config struct {
	ID            string
	Name          string
	Transport     domain.MCPTransport
	Command       string
	Args          []string
	URL           string
	Headers       map[string]string
	SecretHeaders map[string]string
	Env           map[string]string
	SecretEnv     map[string]string
	Timeout       time.Duration
}

// Tool describes an MCP tool returned by tools/list.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// CallResult describes an MCP tools/call result.
type CallResult struct {
	Content []ToolContent   `json:"content,omitempty"`
	IsError bool            `json:"isError,omitempty"`
	Meta    json.RawMessage `json:"_meta,omitempty"`
}

// ToolContent is one content item from a tools/call result.
type ToolContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Data     string          `json:"data,omitempty"`
	MimeType string          `json:"mimeType,omitempty"`
	Raw      json.RawMessage `json:"-"`
}

// JSONRPCRequest is a JSON-RPC 2.0 request envelope.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response envelope.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError is the error object in a JSON-RPC response.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Client is a small MCP JSON-RPC client supporting stdio and HTTP-like transports.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// ConfigFromDomain converts the persisted MCP server configuration to client configuration.
func ConfigFromDomain(cfg domain.MCPServerConfig, defaultTimeout time.Duration) Config {
	timeout := defaultTimeout
	if cfg.TimeoutSec > 0 {
		timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}

	return Config{
		ID:            cfg.ID,
		Name:          cfg.Name,
		Transport:     cfg.Transport,
		Command:       cfg.Command,
		Args:          append([]string(nil), cfg.Args...),
		URL:           cfg.URL,
		Headers:       cloneMap(cfg.Headers),
		SecretHeaders: cloneMap(cfg.SecretHeaders),
		Env:           cloneMap(cfg.Env),
		SecretEnv:     cloneMap(cfg.SecretEnv),
		Timeout:       timeout,
	}
}

// NewClient creates an MCP client and validates the requested transport.
func NewClient(cfg domain.MCPServerConfig, defaultTimeout time.Duration) (*Client, error) {
	clientCfg := ConfigFromDomain(cfg, defaultTimeout)
	if err := validateConfig(clientCfg); err != nil {
		return nil, err
	}

	return &Client{
		cfg: clientCfg,
		httpClient: &http.Client{
			Timeout: clientCfg.Timeout,
		},
	}, nil
}

// Test performs a lightweight connectivity check using tools/list.
func (c *Client) Test(ctx context.Context) error {
	_, err := c.ListTools(ctx)
	return err
}

// ListTools returns the tools advertised by the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var result toolsListResult
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool invokes one MCP tool by name with JSON-compatible arguments.
func (c *Client) CallTool(ctx context.Context, name string, args any) (CallResult, error) {
	if strings.TrimSpace(name) == "" {
		return CallResult{}, errors.New("mcp tool name must not be empty")
	}

	params := toolsCallParams{
		Name:      name,
		Arguments: args,
	}

	var result CallResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return CallResult{}, err
	}
	return result, nil
}

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	request := JSONRPCRequest{
		JSONRPC: jsonrpcVersion,
		ID:      1,
		Method:  method,
		Params:  params,
	}

	var response JSONRPCResponse
	var err error
	switch c.cfg.Transport {
	case domain.MCPTransportStdio:
		response, err = c.callStdio(ctx, request)
	case domain.MCPTransportStreamableHTTP, domain.MCPTransportSSE:
		response, err = c.callHTTP(ctx, request)
	default:
		return fmt.Errorf("unsupported mcp transport %q", c.cfg.Transport)
	}
	if err != nil {
		return err
	}

	if response.Error != nil {
		return fmt.Errorf("mcp json-rpc error %d: %s", response.Error.Code, response.Error.Message)
	}
	if len(response.Result) == 0 {
		return errors.New("mcp json-rpc response missing result")
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("decode mcp %s result: %w", method, err)
	}
	return nil
}

func (c *Client) callStdio(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	payload, err := json.Marshal(request)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("encode mcp stdio request: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.cfg.Command, c.cfg.Args...)
	cmd.Env = mergedEnv(os.Environ(), c.cfg.Env, c.cfg.SecretEnv)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("open mcp stdio stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("open mcp stdio stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return JSONRPCResponse{}, fmt.Errorf("start mcp stdio command: %w", err)
	}

	writeErr := make(chan error, 1)
	go func() {
		_, err := stdin.Write(append(payload, '\n'))
		if closeErr := stdin.Close(); err == nil {
			err = closeErr
		}
		writeErr <- err
	}()

	reader := bufio.NewReader(stdout)
	line, readErr := reader.ReadBytes('\n')
	waitErr := cmd.Wait()
	if writeErrValue := <-writeErr; writeErrValue != nil {
		return JSONRPCResponse{}, fmt.Errorf("write mcp stdio request: %w%s", writeErrValue, c.stderrSuffix(stderr.String()))
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return JSONRPCResponse{}, fmt.Errorf("read mcp stdio response: %w%s", readErr, c.stderrSuffix(stderr.String()))
	}
	if waitErr != nil {
		return JSONRPCResponse{}, fmt.Errorf("mcp stdio command failed: %w%s", waitErr, c.stderrSuffix(stderr.String()))
	}
	if ctx.Err() != nil {
		return JSONRPCResponse{}, fmt.Errorf("mcp stdio command context error: %w%s", ctx.Err(), c.stderrSuffix(stderr.String()))
	}

	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return JSONRPCResponse{}, fmt.Errorf("mcp stdio response was empty%s", c.stderrSuffix(stderr.String()))
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(line, &response); err != nil {
		return JSONRPCResponse{}, fmt.Errorf("decode mcp stdio response: %w%s", err, c.stderrSuffix(stderr.String()))
	}
	return response, nil
}

func (c *Client) callHTTP(ctx context.Context, rpcRequest JSONRPCRequest) (JSONRPCResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	payload, err := json.Marshal(rpcRequest)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("encode mcp http request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("create mcp http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, value := range mergedHeaders(c.cfg.Headers, c.cfg.SecretHeaders) {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("send mcp http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("read mcp http response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return JSONRPCResponse{}, fmt.Errorf("mcp http request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rpcResponse JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return JSONRPCResponse{}, fmt.Errorf("decode mcp http response: %w", err)
	}
	return rpcResponse, nil
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.cfg.Timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.cfg.Timeout)
}

func validateConfig(cfg Config) error {
	if !cfg.Transport.Valid() {
		return fmt.Errorf("mcp transport %q is invalid", cfg.Transport)
	}
	switch cfg.Transport {
	case domain.MCPTransportStdio:
		if strings.TrimSpace(cfg.Command) == "" {
			return errors.New("stdio mcp server command must not be empty")
		}
	case domain.MCPTransportStreamableHTTP, domain.MCPTransportSSE:
		if strings.TrimSpace(cfg.URL) == "" {
			return fmt.Errorf("%s mcp server url must not be empty", cfg.Transport)
		}
	}
	return nil
}

func mergedHeaders(headers map[string]string, secretHeaders map[string]string) map[string]string {
	merged := cloneMap(headers)
	for key, value := range secretHeaders {
		merged[key] = value
	}
	return merged
}

func mergedEnv(base []string, env map[string]string, secretEnv map[string]string) []string {
	merged := make(map[string]string, len(base)+len(env)+len(secretEnv))
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			merged[key] = value
		}
	}
	for key, value := range env {
		merged[key] = value
	}
	for key, value := range secretEnv {
		merged[key] = value
	}

	result := make([]string, 0, len(merged))
	for key, value := range merged {
		result = append(result, key+"="+value)
	}
	return result
}

func cloneMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func (c *Client) stderrSuffix(stderr string) string {
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return ""
	}
	return ": stderr: " + redactSecretValues(trimmed, c.cfg.SecretEnv)
}

func redactSecretValues(message string, secrets map[string]string) string {
	redacted := message
	for _, secret := range secrets {
		if secret != "" {
			redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")
		}
	}
	return redacted
}

type toolsListResult struct {
	Tools []Tool `json:"tools"`
}

type toolsCallParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments,omitempty"`
}
