//go:build eval

package main

import (
	"github.com/alexmdac/starlark-mcp/internal/llm"
)

// responseToHistory converts an LLM response into a Message suitable for
// appending to the conversation history.
func responseToHistory(resp *llm.MessageResponse) llm.Message {
	return llm.Message{
		Role:      llm.RoleAssistant,
		Text:      resp.Text,
		ToolCalls: resp.ToolCalls,
	}
}

// toolResultMessage creates a user message carrying a tool result.
func toolResultMessage(toolCallID, content string, isError bool) llm.Message {
	return llm.Message{
		Role: llm.RoleUser,
		ToolResult: &llm.ToolResult{
			ToolCallID: toolCallID,
			Content:    content,
			IsError:    isError,
		},
	}
}

// toolResultWithNudge creates a user message with a tool result and a text nudge.
func toolResultWithNudge(toolCallID, content, nudge string) llm.Message {
	return llm.Message{
		Role: llm.RoleUser,
		ToolResult: &llm.ToolResult{
			ToolCallID: toolCallID,
			Content:    content,
			IsError:    false,
		},
		Text: nudge,
	}
}
