package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOllama_Defaults(t *testing.T) {
	c := NewOllama("llama3", "")
	if c.OpenAIClient.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %q, want default ollama URL", c.OpenAIClient.BaseURL)
	}
	if c.OpenAIClient.Model != "llama3" {
		t.Errorf("Model = %q, want llama3", c.OpenAIClient.Model)
	}
	if c.OpenAIClient.APIKey != "ollama" {
		t.Errorf("APIKey = %q, want ollama", c.OpenAIClient.APIKey)
	}
}

func TestNewOllama_CustomBaseURL(t *testing.T) {
	c := NewOllama("mistral", "http://myhost:9999")
	if c.OpenAIClient.BaseURL != "http://myhost:9999" {
		t.Errorf("BaseURL = %q, want custom URL", c.OpenAIClient.BaseURL)
	}
}

func TestOllamaSendMessage_SynthesizesToolCallIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{
							ID:   "", // Ollama may return empty IDs
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "execute-starlark",
								Arguments: `{"program":"print(1)"}`,
							},
						},
						{
							ID:   "",
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "another-tool",
								Arguments: `{}`,
							},
						},
					},
				}},
			},
			Usage: openAIUsage{PromptTokens: 10, CompletionTokens: 5},
		})
	}))
	defer srv.Close()

	c := NewOllama("llama3", srv.URL)
	resp, err := c.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "run it"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Fatalf("tool calls = %d, want 2", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "ollama_call_0" {
		t.Errorf("tool call 0 ID = %q, want ollama_call_0", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[1].ID != "ollama_call_1" {
		t.Errorf("tool call 1 ID = %q, want ollama_call_1", resp.ToolCalls[1].ID)
	}
}

func TestOllamaSendMessage_PreservesExistingIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{
							ID:   "real_id_123",
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "execute-starlark",
								Arguments: `{"program":"print(1)"}`,
							},
						},
					},
				}},
			},
		})
	}))
	defer srv.Close()

	c := NewOllama("llama3", srv.URL)
	resp, err := c.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "run it"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if resp.ToolCalls[0].ID != "real_id_123" {
		t.Errorf("tool call ID = %q, want real_id_123 (should not be overwritten)", resp.ToolCalls[0].ID)
	}
}

func TestOllamaSendMessage_TextOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "Hello!"}},
			},
			Usage: openAIUsage{PromptTokens: 5, CompletionTokens: 3},
		})
	}))
	defer srv.Close()

	c := NewOllama("llama3", srv.URL)
	resp, err := c.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if resp.Text != "Hello!" {
		t.Errorf("text = %q, want Hello!", resp.Text)
	}
}
