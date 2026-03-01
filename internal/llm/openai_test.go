package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAISendMessage_TextOnly(t *testing.T) {
	var gotReq openAIRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "Hello back!"}},
			},
			Usage: openAIUsage{PromptTokens: 10, CompletionTokens: 5},
		})
	}))
	defer srv.Close()

	p := NewOpenAI("test-key", "gpt-test", srv.URL, ClientOpts{})
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
	if gotReq.Model != "gpt-test" {
		t.Errorf("model = %q, want %q", gotReq.Model, "gpt-test")
	}
	if gotReq.MaxCompletionTokens != 100 {
		t.Errorf("max_completion_tokens = %d, want 100", gotReq.MaxCompletionTokens)
	}
	if len(gotReq.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2 (system + user)", len(gotReq.Messages))
	}
	if gotReq.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want system", gotReq.Messages[0].Role)
	}
	if gotReq.Messages[0].Content != "Be helpful." {
		t.Errorf("system content = %q", gotReq.Messages[0].Content)
	}
	if gotReq.Messages[1].Role != "user" {
		t.Errorf("second message role = %q, want user", gotReq.Messages[1].Role)
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

func TestOpenAISendMessage_ToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{
							ID:   "call_123",
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

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
	resp, err := p.SendMessage(context.Background(), &MessageParams{
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
	if tc.ID != "call_123" {
		t.Errorf("tool call ID = %q", tc.ID)
	}
	if tc.Name != "execute-starlark" {
		t.Errorf("tool call name = %q", tc.Name)
	}
	var input map[string]any
	if err := json.Unmarshal(tc.Input, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input["program"] != "print(1)" {
		t.Errorf("input program = %v", input["program"])
	}
}

func TestOpenAISendMessage_ToolDefs(t *testing.T) {
	var gotReq openAIRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
		})
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
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
	if tool.Type != "function" {
		t.Errorf("type = %q, want function", tool.Type)
	}
	if tool.Function.Name != "execute-starlark" {
		t.Errorf("name = %q", tool.Function.Name)
	}
	if tool.Function.Description != "Run a Starlark program" {
		t.Errorf("description = %q", tool.Function.Description)
	}
	if tool.Function.Parameters["type"] != "object" {
		t.Errorf("parameters type = %v", tool.Function.Parameters["type"])
	}
	props, ok := tool.Function.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties not a map")
	}
	if _, ok := props["program"]; !ok {
		t.Errorf("missing 'program' in properties")
	}
}

func TestOpenAISendMessage_AssistantTextWithToolCalls(t *testing.T) {
	var gotReq openAIRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
		})
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
	_, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages: []Message{
			{Role: RoleUser, Text: "run it"},
			{
				Role:      RoleAssistant,
				Text:      "I'll run that for you.",
				ToolCalls: []ToolCall{{ID: "t1", Name: "foo", Input: json.RawMessage(`{}`)}},
			},
			{
				Role:       RoleUser,
				ToolResult: &ToolResult{ToolCallID: "t1", Content: "42"},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	// Message 1 should be the assistant with both text and tool_calls.
	msg := gotReq.Messages[1]
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want assistant", msg.Role)
	}
	if msg.Content != "I'll run that for you." {
		t.Errorf("content = %q, want assistant text preserved", msg.Content)
	}
	if len(msg.ToolCalls) != 1 {
		t.Errorf("tool_calls = %d, want 1", len(msg.ToolCalls))
	}
}

func TestOpenAISendMessage_ToolResult(t *testing.T) {
	var gotReq openAIRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
		})
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
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

	// system is not set, so: user, assistant (tool_calls), tool, user (nudge) = 4
	if len(gotReq.Messages) != 4 {
		for i, m := range gotReq.Messages {
			t.Logf("msg[%d]: role=%s content=%q tool_call_id=%q tool_calls=%d",
				i, m.Role, m.Content, m.ToolCallID, len(m.ToolCalls))
		}
		t.Fatalf("messages len = %d, want 4", len(gotReq.Messages))
	}

	// Message 0: user
	if gotReq.Messages[0].Role != "user" {
		t.Errorf("msg[0] role = %q", gotReq.Messages[0].Role)
	}

	// Message 1: assistant with tool_calls
	if gotReq.Messages[1].Role != "assistant" {
		t.Errorf("msg[1] role = %q", gotReq.Messages[1].Role)
	}
	if len(gotReq.Messages[1].ToolCalls) != 1 {
		t.Errorf("msg[1] tool_calls = %d", len(gotReq.Messages[1].ToolCalls))
	}

	// Message 2: tool result
	if gotReq.Messages[2].Role != "tool" {
		t.Errorf("msg[2] role = %q, want tool", gotReq.Messages[2].Role)
	}
	if gotReq.Messages[2].ToolCallID != "t1" {
		t.Errorf("msg[2] tool_call_id = %q", gotReq.Messages[2].ToolCallID)
	}
	if gotReq.Messages[2].Content != "42" {
		t.Errorf("msg[2] content = %q", gotReq.Messages[2].Content)
	}

	// Message 3: user nudge
	if gotReq.Messages[3].Role != "user" {
		t.Errorf("msg[3] role = %q, want user", gotReq.Messages[3].Role)
	}
	if gotReq.Messages[3].Content != "Try again." {
		t.Errorf("msg[3] content = %q", gotReq.Messages[3].Content)
	}
}

func TestOpenAISendMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"server error"}}`))
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
	_, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !contains(got, "500") {
		t.Errorf("error = %q, want to contain 500", got)
	}
}

func TestOpenAISendMessage_NoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{Choices: []openAIChoice{}})
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
	_, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestOpenAISendMessage_RetryAfter(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
			Usage: openAIUsage{PromptTokens: 5, CompletionTokens: 3},
		})
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
	resp, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if resp.Text != "ok" {
		t.Errorf("text = %q, want ok", resp.Text)
	}
}

func TestOpenAISendMessage_429BackoffWithoutHeader(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
		})
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
	p.InitialBackoff = 1 * time.Millisecond
	resp, err := p.SendMessage(context.Background(), &MessageParams{
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Text: "hi"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if resp.Text != "ok" {
		t.Errorf("text = %q, want ok", resp.Text)
	}
}

func TestOpenAISendMessage_429ExhaustedRetries(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	p := NewOpenAI("k", "m", srv.URL, ClientOpts{})
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
	if attempts != maxRetries {
		t.Errorf("attempts = %d, want %d (maxRetries)", attempts, maxRetries)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"", -1},
		{"0", 0},
		{"1", 1 * time.Second},
		{"30", 30 * time.Second},
		{"-1", -1},
		{"not-a-number", -1},
	}
	for _, tt := range tests {
		if got := parseRetryAfter(tt.input); got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
