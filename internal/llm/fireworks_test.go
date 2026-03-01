package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewFireworks(t *testing.T) {
	c := NewFireworks("fw-key", "accounts/fireworks/models/deepseek-v3p2", "https://api.fireworks.ai")
	if c.OpenAIClient.BaseURL != "https://api.fireworks.ai" {
		t.Errorf("BaseURL = %q, want https://api.fireworks.ai", c.OpenAIClient.BaseURL)
	}
	if c.OpenAIClient.Model != "accounts/fireworks/models/deepseek-v3p2" {
		t.Errorf("Model = %q, want accounts/fireworks/models/deepseek-v3p2", c.OpenAIClient.Model)
	}
	if c.OpenAIClient.APIKey != "fw-key" {
		t.Errorf("APIKey = %q, want fw-key", c.OpenAIClient.APIKey)
	}
}

func TestFireworksSendMessage_TextOnly(t *testing.T) {
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

	c := NewFireworks("fw-key", "accounts/fireworks/models/deepseek-v3p2", srv.URL)
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
	if resp.Usage.InputTokens != 5 {
		t.Errorf("input tokens = %d, want 5", resp.Usage.InputTokens)
	}
}

func TestFireworksSendMessage_ToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{
							ID:   "call_fw_1",
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "execute-starlark",
								Arguments: `{"program":"print(1)"}`,
							},
						},
					},
				}},
			},
			Usage: openAIUsage{PromptTokens: 20, CompletionTokens: 15},
		})
	}))
	defer srv.Close()

	c := NewFireworks("fw-key", "accounts/fireworks/models/deepseek-v3p2", srv.URL)
	resp, err := c.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "run it"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_fw_1" {
		t.Errorf("tool call ID = %q, want call_fw_1", tc.ID)
	}
	if tc.Name != "execute-starlark" {
		t.Errorf("tool call name = %q, want execute-starlark", tc.Name)
	}
}
