// Package llm provides a provider-agnostic interface for LLM chat completions.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Client sends a message to an LLM and returns the response.
type Client interface {
	SendMessage(ctx context.Context, params *MessageParams) (*MessageResponse, error)
}

// MessageParams describes a request to an LLM.
type MessageParams struct {
	System    string
	Messages  []Message
	Tools     []ToolDef
	MaxTokens int
}

// Message is a single message in the conversation.
type Message struct {
	Role       Role
	Text       string      // for user/assistant text
	ToolCalls  []ToolCall  // for assistant messages requesting tool use
	ToolResult *ToolResult // for user messages returning tool results
}

// Role identifies the sender of a message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// ToolCall represents the LLM requesting a tool invocation.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult is the outcome of executing a tool call.
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

// ToolDef describes a tool available to the LLM.
type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// MessageResponse is the LLM's reply.
type MessageResponse struct {
	Text      string     // text content (may be empty if only tool calls)
	ToolCalls []ToolCall // tool calls requested by the model
	Usage     Usage
}

// Usage reports token consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// ClientOpts holds optional configuration for LLM clients.
type ClientOpts struct {
	// RequestTimeout is the timeout for each individual LLM HTTP request.
	// Zero means no timeout.
	RequestTimeout time.Duration
}

// ParseModel parses a "provider:model" string.
// The provider prefix is always required.
func ParseModel(s string) (provider, model string, err error) {
	i := strings.Index(s, ":")
	if i < 0 {
		return "", "", fmt.Errorf("model %q must have a provider prefix (providers: anthropic, openai, fireworks, ollama)", s)
	}
	return s[:i], s[i+1:], nil
}
