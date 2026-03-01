package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicSendMessage_TextOnly(t *testing.T) {
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected api key: %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("unexpected version header: %s", r.Header.Get("anthropic-version"))
		}

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			Content: []anthropicContentBlock{
				{Type: "text", Text: "Hello back!"},
			},
			Usage: anthropicUsage{InputTokens: 10, OutputTokens: 5},
		})
	}))
	defer srv.Close()

	p := NewAnthropic("test-key", "claude-test", srv.URL, ClientOpts{})
	resp, err := p.SendMessage(context.Background(), &MessageParams{
		System:    "Be helpful.",
		MaxTokens: 100,
		Messages: []Message{
			{Role: RoleUser, Text: "Hello!"},
		},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	// Verify request was built correctly.
	if gotReq.Model != "claude-test" {
		t.Errorf("model = %q, want %q", gotReq.Model, "claude-test")
	}
	if gotReq.System != "Be helpful." {
		t.Errorf("system = %q, want %q", gotReq.System, "Be helpful.")
	}
	if gotReq.MaxTokens != 100 {
		t.Errorf("max_tokens = %d, want 100", gotReq.MaxTokens)
	}
	if len(gotReq.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(gotReq.Messages))
	}

	// Verify response was parsed correctly.
	if resp.Text != "Hello back!" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello back!")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("tool calls = %d, want 0", len(resp.ToolCalls))
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("input tokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("output tokens = %d, want 5", resp.Usage.OutputTokens)
	}
}

func TestAnthropicSendMessage_ToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			Content: []anthropicContentBlock{
				{Type: "text", Text: "I'll run that."},
				{
					Type:  "tool_use",
					ID:    "toolu_123",
					Name:  "execute-starlark",
					Input: json.RawMessage(`{"program":"print(1)"}`),
				},
			},
			Usage: anthropicUsage{InputTokens: 20, OutputTokens: 15},
		})
	}))
	defer srv.Close()

	p := NewAnthropic("k", "m", srv.URL, ClientOpts{})
	resp, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "run it"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if resp.Text != "I'll run that." {
		t.Errorf("text = %q", resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_123" {
		t.Errorf("tool call ID = %q", tc.ID)
	}
	if tc.Name != "execute-starlark" {
		t.Errorf("tool call name = %q", tc.Name)
	}
}

func TestAnthropicSendMessage_ToolResult(t *testing.T) {
	// Verify that tool result messages are serialized correctly.
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			Content: []anthropicContentBlock{{Type: "text", Text: "ok"}},
			Usage:   anthropicUsage{},
		})
	}))
	defer srv.Close()

	p := NewAnthropic("k", "m", srv.URL, ClientOpts{})
	_, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages: []Message{
			{Role: RoleUser, Text: "run it"},
			{
				Role:      RoleAssistant,
				ToolCalls: []ToolCall{{ID: "t1", Name: "foo", Input: json.RawMessage(`{"x":1}`)}},
			},
			{
				Role:       RoleUser,
				ToolResult: &ToolResult{ToolCallID: "t1", Content: "42", IsError: false},
				Text:       "Try again.",
			},
		},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if len(gotReq.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(gotReq.Messages))
	}

	// The tool result message should have both tool_result and text blocks.
	trMsg := gotReq.Messages[2]
	if trMsg.Role != "user" {
		t.Errorf("role = %q, want user", trMsg.Role)
	}
	if len(trMsg.Content) != 2 {
		t.Fatalf("content blocks = %d, want 2", len(trMsg.Content))
	}
	if trMsg.Content[0]["type"] != "tool_result" {
		t.Errorf("first block type = %v", trMsg.Content[0]["type"])
	}
	if trMsg.Content[1]["type"] != "text" {
		t.Errorf("second block type = %v", trMsg.Content[1]["type"])
	}
}

func TestAnthropicSendMessage_ToolDefs(t *testing.T) {
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			Content: []anthropicContentBlock{{Type: "text", Text: "ok"}},
		})
	}))
	defer srv.Close()

	p := NewAnthropic("k", "m", srv.URL, ClientOpts{})
	_, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
		Tools: []ToolDef{
			{
				Name:        "execute-starlark",
				Description: "Run a Starlark program",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"program": map[string]any{"type": "string"},
					},
					"required": []any{"program"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if len(gotReq.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(gotReq.Tools))
	}
	tool := gotReq.Tools[0]
	if tool.Name != "execute-starlark" {
		t.Errorf("name = %q", tool.Name)
	}
	if tool.Description != "Run a Starlark program" {
		t.Errorf("description = %q", tool.Description)
	}
	if tool.InputSchema["type"] != "object" {
		t.Errorf("input_schema type = %v", tool.InputSchema["type"])
	}
	props, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties not a map")
	}
	if _, ok := props["program"]; !ok {
		t.Errorf("missing 'program' in properties")
	}
}

func TestAnthropicSendMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	p := NewAnthropic("k", "m", srv.URL, ClientOpts{})
	_, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !contains(got, "429") {
		t.Errorf("error = %q, want to contain 429", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
