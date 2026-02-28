//go:build eval

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Content block helpers using map[string]any to avoid JSON polymorphism issues.

func textBlock(text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
	}
}

func toolUseBlock(id, name string, input map[string]any) map[string]any {
	return map[string]any{
		"type":  "tool_use",
		"id":    id,
		"name":  name,
		"input": input,
	}
}

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

// apiMessage represents a message in the conversation.
type apiMessage struct {
	Role    string           `json:"role"`
	Content []map[string]any `json:"content"`
}

// apiToolDef represents a tool definition for the API.
type apiToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// apiRequest represents a request to the Anthropic Messages API.
type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Tools     []apiToolDef `json:"tools,omitempty"`
}

// apiResponseContentBlock represents a content block in the API response.
type apiResponseContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// apiResponseUsage represents token usage in the API response.
type apiResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// apiResponse represents a response from the Anthropic Messages API.
type apiResponse struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Role       string                    `json:"role"`
	Content    []apiResponseContentBlock `json:"content"`
	StopReason string                    `json:"stop_reason"`
	Usage      apiResponseUsage          `json:"usage"`
}

// responseToMessage converts an API response into an apiMessage suitable for
// appending to the conversation history.
func responseToMessage(resp *apiResponse) apiMessage {
	blocks := make([]map[string]any, len(resp.Content))
	for i, cb := range resp.Content {
		switch cb.Type {
		case "text":
			blocks[i] = textBlock(cb.Text)
		case "tool_use":
			var input map[string]any
			_ = json.Unmarshal(cb.Input, &input)
			blocks[i] = toolUseBlock(cb.ID, cb.Name, input)
		default:
			blocks[i] = map[string]any{"type": cb.Type}
		}
	}
	return apiMessage{
		Role:    resp.Role,
		Content: blocks,
	}
}

// toolInput represents the parsed input for the execute-starlark tool.
type toolInput struct {
	Program     string  `json:"program"`
	TimeoutSecs float64 `json:"timeout_secs"`
}

// llmClient is a simple client for the Anthropic Messages API.
type llmClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func newLLMClient(apiKey, model string) *llmClient {
	return &llmClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{},
	}
}

func (c *llmClient) sendRequest(ctx context.Context, req *apiRequest) (*apiResponse, error) {
	req.Model = c.model

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
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

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &apiResp, nil
}
