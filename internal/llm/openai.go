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

// OpenAIProvider implements Provider for the OpenAI Chat Completions API
// and any compatible endpoint.
type OpenAIProvider struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
	HTTP    *http.Client
}

// NewOpenAI creates an OpenAI-compatible provider.
func NewOpenAI(apiKey, model, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Timeout: 120 * time.Second,
		HTTP:    &http.Client{},
	}
}

// SendMessage implements Provider.
func (p *OpenAIProvider) SendMessage(ctx context.Context, params *MessageParams) (*MessageResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	req := p.buildRequest(params)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

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

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return p.parseResponse(&apiResp)
}

// --- OpenAI wire types ---

type openAIRequest struct {
	Model    string            `json:"model"`
	Messages []openAIMessage   `json:"messages"`
	Tools    []openAIToolDef   `json:"tools,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []openAIToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function openAIFunctionCall  `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIToolDef struct {
	Type     string           `json:"type"`
	Function openAIFunctionDef `json:"function"`
}

type openAIFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// --- conversion helpers ---

func (p *OpenAIProvider) buildRequest(params *MessageParams) *openAIRequest {
	var messages []openAIMessage

	// System message.
	if params.System != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: params.System,
		})
	}

	for _, m := range params.Messages {
		messages = append(messages, toOpenAIMessages(m)...)
	}

	tools := make([]openAIToolDef, len(params.Tools))
	for i, t := range params.Tools {
		tools[i] = openAIToolDef{
			Type: "function",
			Function: openAIFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
	}

	return &openAIRequest{
		Model:     p.Model,
		Messages:  messages,
		Tools:     tools,
		MaxTokens: params.MaxTokens,
	}
}

// toOpenAIMessages converts a single llm.Message into one or more OpenAI messages.
// A tool result becomes a "tool" role message. If the same Message also has text
// (e.g. a nudge), it becomes a separate "user" message.
func toOpenAIMessages(m Message) []openAIMessage {
	var out []openAIMessage

	if m.ToolResult != nil {
		out = append(out, openAIMessage{
			Role:       "tool",
			Content:    m.ToolResult.Content,
			ToolCallID: m.ToolResult.ToolCallID,
		})
	}

	if len(m.ToolCalls) > 0 {
		tcs := make([]openAIToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			tcs[i] = openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openAIFunctionCall{
					Name:      tc.Name,
					Arguments: string(tc.Input),
				},
			}
		}
		out = append(out, openAIMessage{
			Role:      "assistant",
			ToolCalls: tcs,
		})
	} else if m.Text != "" && m.ToolResult == nil {
		// Plain text message (user or assistant).
		out = append(out, openAIMessage{
			Role:    string(m.Role),
			Content: m.Text,
		})
	}

	// Text attached to a tool result (nudge) becomes a separate user message.
	if m.Text != "" && m.ToolResult != nil {
		out = append(out, openAIMessage{
			Role:    "user",
			Content: m.Text,
		})
	}

	return out
}

func (p *OpenAIProvider) parseResponse(resp *openAIResponse) (*MessageResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	msg := resp.Choices[0].Message
	result := &MessageResponse{
		Text: msg.Content,
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	for _, tc := range msg.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	return result, nil
}
