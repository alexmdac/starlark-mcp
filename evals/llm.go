package evals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Content block helpers using map[string]any to avoid JSON polymorphism issues.

// TextBlock creates a text content block.
func TextBlock(text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
	}
}

// ToolUseBlock creates a tool_use content block.
func ToolUseBlock(id, name string, input map[string]any) map[string]any {
	return map[string]any{
		"type":  "tool_use",
		"id":    id,
		"name":  name,
		"input": input,
	}
}

// ToolResultBlock creates a tool_result content block.
func ToolResultBlock(toolUseID, content string, isError bool) map[string]any {
	b := map[string]any{
		"type":        "tool_result",
		"tool_use_id": toolUseID,
		"content":     content,
	}
	if isError {
		b["is_error"] = true
	}
	return b
}

// Message represents a message in the conversation.
type Message struct {
	Role    string           `json:"role"`
	Content []map[string]any `json:"content"`
}

// ToolDef represents a tool definition for the API.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Request represents a request to the Anthropic Messages API.
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []ToolDef `json:"tools,omitempty"`
}

// ResponseContentBlock represents a content block in the API response.
type ResponseContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ResponseUsage represents token usage in the API response.
type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Response represents a response from the Anthropic Messages API.
type Response struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Role       string                 `json:"role"`
	Content    []ResponseContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
	Usage      ResponseUsage          `json:"usage"`
}

// ResponseToMessage converts an API response into a Message suitable for
// appending to the conversation history.
func ResponseToMessage(resp *Response) Message {
	blocks := make([]map[string]any, len(resp.Content))
	for i, cb := range resp.Content {
		switch cb.Type {
		case "text":
			blocks[i] = TextBlock(cb.Text)
		case "tool_use":
			var input map[string]any
			_ = json.Unmarshal(cb.Input, &input)
			blocks[i] = ToolUseBlock(cb.ID, cb.Name, input)
		default:
			blocks[i] = map[string]any{"type": cb.Type}
		}
	}
	return Message{
		Role:    resp.Role,
		Content: blocks,
	}
}

// ToolInput represents the parsed input for the execute-starlark tool.
type ToolInput struct {
	Program     string  `json:"program"`
	TimeoutSecs float64 `json:"timeout_secs"`
}

// Client is a simple client for the Anthropic Messages API.
type Client struct {
	apiKey  string
	model   string
	baseURL string
	timeout time.Duration
	http    *http.Client
}

// NewClient creates a new LLM client.
func NewClient(apiKey, model, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		timeout: 120 * time.Second,
		http:    &http.Client{},
	}
}

// SendRequest sends a request to the Anthropic Messages API.
func (c *Client) SendRequest(ctx context.Context, req *Request) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req.Model = c.model

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &apiResp, nil
}
