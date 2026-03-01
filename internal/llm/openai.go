package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// OpenAIClient implements Client for the OpenAI Chat Completions API
// and any compatible endpoint.
type OpenAIClient struct {
	APIKey         string
	Model          string
	BaseURL        string
	Timeout        time.Duration
	HTTP           *http.Client
	InitialBackoff time.Duration // initial retry delay on 429; 0 uses default (2s)
}

// NewOpenAI creates an OpenAI-compatible client.
func NewOpenAI(apiKey, model, baseURL string) *OpenAIClient {
	return &OpenAIClient{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Timeout: 120 * time.Second,
		HTTP:    &http.Client{},
	}
}

// SendMessage implements Client.
func (p *OpenAIClient) SendMessage(ctx context.Context, params *MessageParams) (*MessageResponse, error) {
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, respBody, err := p.doWithRetry(ctx, httpReq, body)
	if err != nil {
		return nil, err
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

const maxRetries = 8

// doWithRetry sends an HTTP request and retries on 429 with exponential backoff
// plus jitter. It respects the Retry-After header when present, otherwise uses
// exponential backoff starting from InitialBackoff (default 2s).
func (p *OpenAIClient) doWithRetry(ctx context.Context, req *http.Request, body []byte) (*http.Response, []byte, error) {
	backoff := p.InitialBackoff
	if backoff <= 0 {
		backoff = 2 * time.Second
	}

	for attempt := range maxRetries {
		var httpReq *http.Request
		if attempt == 0 {
			httpReq = req
		} else {
			var err error
			httpReq, err = http.NewRequestWithContext(ctx, req.Method, req.URL.String(), bytes.NewReader(body))
			if err != nil {
				return nil, nil, fmt.Errorf("create retry request: %w", err)
			}
			httpReq.Header = req.Header
		}

		resp, respBody, err := p.doRequest(httpReq)
		if err != nil {
			return nil, nil, err
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, respBody, nil
		}

		// Last attempt — return the 429 as-is.
		if attempt == maxRetries-1 {
			return resp, respBody, nil
		}

		delay := parseRetryAfter(resp.Header.Get("Retry-After"))
		if delay < 0 {
			// Add jitter: backoff + rand(0, backoff/2) to avoid thundering herd.
			jitter := time.Duration(rand.Int64N(int64(backoff / 2)))
			delay = backoff + jitter
			backoff *= 2
		}

		if delay == 0 {
			continue
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}

	// unreachable
	return nil, nil, fmt.Errorf("unexpected: exceeded max retries")
}

// doRequest executes a single HTTP request and returns the response with body read.
func (p *OpenAIClient) doRequest(req *http.Request) (*http.Response, []byte, error) {
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}
	return resp, body, nil
}

// parseRetryAfter parses a Retry-After header value as seconds.
// Only the integer-seconds format is supported; HTTP-date is not.
// Returns -1 if the header is missing or unparseable.
func parseRetryAfter(s string) time.Duration {
	if s == "" {
		return -1
	}
	secs, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	if secs < 0 {
		return -1
	}
	return time.Duration(secs) * time.Second
}

// --- OpenAI wire types ---

type openAIRequest struct {
	Model               string          `json:"model"`
	Messages            []openAIMessage `json:"messages"`
	Tools               []openAIToolDef `json:"tools,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	ParallelToolCalls   *bool           `json:"parallel_tool_calls,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIToolDef struct {
	Type     string            `json:"type"`
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

func (p *OpenAIClient) buildRequest(params *MessageParams) *openAIRequest {
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

	parallelToolCalls := false
	return &openAIRequest{
		Model:               p.Model,
		Messages:            messages,
		Tools:               tools,
		MaxCompletionTokens: params.MaxTokens,
		ParallelToolCalls:   &parallelToolCalls,
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
			Content:   m.Text,
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

func (p *OpenAIClient) parseResponse(resp *openAIResponse) (*MessageResponse, error) {
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
