package llm

import (
	"net/http"
	"time"
)

// FireworksClient implements Client for the Fireworks AI inference API.
// Fireworks exposes an OpenAI-compatible API, so this wraps OpenAIClient
// with Fireworks-appropriate defaults. It exists as a separate type so that
// Fireworks-specific behavior (e.g. auth, model aliases) can be added later.
type FireworksClient struct {
	*OpenAIClient
}

// NewFireworks creates a client for the Fireworks AI API.
func NewFireworks(apiKey, model, baseURL string) *FireworksClient {
	return &FireworksClient{
		OpenAIClient: &OpenAIClient{
			APIKey:  apiKey,
			Model:   model,
			BaseURL: baseURL,
			Timeout: 120 * time.Second,
			HTTP:    &http.Client{},
		},
	}
}
