package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicClient implements Client for the Anthropic Messages API.
type AnthropicClient struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
	HTTP    *http.Client
}

// NewAnthropic creates an Anthropic client.
func NewAnthropic(apiKey, model, baseURL string) *AnthropicClient {
	return &AnthropicClient{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Timeout: 120 * time.Second,
		HTTP:    &http.Client{},
	}
}

// SendMessage implements Client.
func (p *AnthropicClient) SendMessage(ctx context.Context, params *MessageParams) (*MessageResponse, error) {
	if p.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.Timeout)
		defer cancel()
	}

	req := p.buildRequest(params)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := p.HTTP.Do(httpReq)
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

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return p.parseResponse(&apiResp), nil
}

// --- Anthropic wire types ---

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicToolDef `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string           `json:"role"`
	Content []map[string]any `json:"content"`
}

type anthropicToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Usage   anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// --- conversion helpers ---

func (p *AnthropicClient) buildRequest(params *MessageParams) *anthropicRequest {
	messages := make([]anthropicMessage, len(params.Messages))
	for i, m := range params.Messages {
		messages[i] = toAnthropicMessage(m)
	}

	tools := make([]anthropicToolDef, len(params.Tools))
	for i, t := range params.Tools {
		tools[i] = anthropicToolDef(t)
	}

	return &anthropicRequest{
		Model:     p.Model,
		MaxTokens: params.MaxTokens,
		System:    params.System,
		Messages:  messages,
		Tools:     tools,
	}
}

func toAnthropicMessage(m Message) anthropicMessage {
	var blocks []map[string]any

	if m.ToolResult != nil {
		b := map[string]any{
			"type":        "tool_result",
			"tool_use_id": m.ToolResult.ToolCallID,
			"content":     m.ToolResult.Content,
		}
		if m.ToolResult.IsError {
			b["is_error"] = true
		}
		blocks = append(blocks, b)
	}

	if m.Text != "" {
		blocks = append(blocks, map[string]any{
			"type": "text",
			"text": m.Text,
		})
	}

	for _, tc := range m.ToolCalls {
		var input map[string]any
		_ = json.Unmarshal(tc.Input, &input)
		blocks = append(blocks, map[string]any{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": input,
		})
	}

	return anthropicMessage{
		Role:    string(m.Role),
		Content: blocks,
	}
}

func (p *AnthropicClient) parseResponse(resp *anthropicResponse) *MessageResponse {
	result := &MessageResponse{
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	for _, cb := range resp.Content {
		switch cb.Type {
		case "text":
			result.Text += cb.Text
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:    cb.ID,
				Name:  cb.Name,
				Input: cb.Input,
			})
		}
	}

	return result
}
