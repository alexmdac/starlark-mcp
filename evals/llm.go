//go:build eval

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Content blocks are represented as map[string]any because the Anthropic API
// uses a union type (text, tool_use, tool_result) that would require custom
// JSON marshaling to model with Go types.

// textBlock creates a text content block.
func textBlock(text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
	}
}

// toolUseBlock creates a tool_use content block.
func toolUseBlock(id, name string, input map[string]any) map[string]any {
	return map[string]any{
		"type":  "tool_use",
		"id":    id,
		"name":  name,
		"input": input,
	}
}

// toolResultBlock creates a tool_result content block.
func toolResultBlock(toolUseID, content string, isError bool) map[string]any {
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

// message represents a message in the conversation.
type message struct {
	Role    string           `json:"role"`
	Content []map[string]any `json:"content"`
}

// toolDef represents a tool definition for the API.
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// request represents a request to the Anthropic Messages API.
type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
	Tools     []toolDef `json:"tools,omitempty"`
}

// responseContentBlock represents a content block in the API response.
type responseContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// responseUsage represents token usage in the API response.
type responseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// response represents a response from the Anthropic Messages API.
type response struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Role       string                 `json:"role"`
	Content    []responseContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
	Usage      responseUsage          `json:"usage"`
}

// responseToMessage converts an API response into a message suitable for
// appending to the conversation history.
func responseToMessage(resp *response) message {
	blocks := make([]map[string]any, len(resp.Content))
	for i, cb := range resp.Content {
		switch cb.Type {
		case "text":
			blocks[i] = textBlock(cb.Text)
		case "tool_use":
			var input map[string]any
			// Unmarshal error is intentionally ignored: if the LLM returns
			// malformed JSON, input stays nil which is safe for the
			// conversation history (the next tool call will fail clearly).
			_ = json.Unmarshal(cb.Input, &input)
			blocks[i] = toolUseBlock(cb.ID, cb.Name, input)
		default:
			blocks[i] = map[string]any{"type": cb.Type}
		}
	}
	return message{
		Role:    resp.Role,
		Content: blocks,
	}
}

// client is a simple client for the Anthropic Messages API.
type client struct {
	apiKey  string
	model   string
	baseURL string
	timeout time.Duration
	http    *http.Client
}

// newClient creates a new LLM client.
func newClient(apiKey, model, baseURL string) *client {
	return &client{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		timeout: 120 * time.Second,
		http:    &http.Client{},
	}
}

// sendRequest sends a request to the Anthropic Messages API.
func (c *client) sendRequest(ctx context.Context, req *request) (*response, error) {
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

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &apiResp, nil
}
