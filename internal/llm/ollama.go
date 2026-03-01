package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const defaultOllamaURL = "http://localhost:11434"

// OllamaClient implements Client for local Ollama models.
// Ollama exposes an OpenAI-compatible API, so this wraps OpenAIClient
// with Ollama-appropriate defaults.
type OllamaClient struct {
	*OpenAIClient
}

// NewOllama creates a client for a local Ollama instance.
// If baseURL is empty, it defaults to http://localhost:11434.
func NewOllama(model, baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}
	return &OllamaClient{
		OpenAIClient: &OpenAIClient{
			APIKey:  "ollama", // Ollama doesn't require an API key but the field must be non-empty.
			Model:   model,
			BaseURL: baseURL,
			Timeout: 300 * time.Second, // Local models can be slow, especially on first load.
			HTTP:    &http.Client{},
		},
	}
}

// SendMessage implements Client. It delegates to the OpenAI-compatible
// endpoint and patches up Ollama-specific quirks (e.g. missing tool call IDs).
func (c *OllamaClient) SendMessage(ctx context.Context, params *MessageParams) (*MessageResponse, error) {
	resp, err := c.OpenAIClient.SendMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Ollama may return empty tool call IDs; synthesize them so downstream
	// code that relies on IDs (e.g. tool result matching) still works.
	for i := range resp.ToolCalls {
		if resp.ToolCalls[i].ID == "" {
			resp.ToolCalls[i].ID = fmt.Sprintf("ollama_call_%d", i)
		}
	}

	return resp, nil
}
