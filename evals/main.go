//go:build eval

// Command eval runs the LLM eval harness against the Starlark MCP server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/alexmdac/starlark-mcp/server"
)

type evalResult struct {
	Case      Case
	Passed    bool
	Attempts  int
	Score     float64
	Outputs   []string // starlark output from each attempt
	TokensIn  int
	TokensOut int
}

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY is required")
		os.Exit(1)
	}

	model := os.Getenv("EVAL_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	client := NewClient(apiKey, model, baseURL)

	results := make([]evalResult, len(Cases))

	// Run cases sequentially (simple; parallelism can be added later with goroutines).
	for i, ec := range Cases {
		fmt.Fprintf(os.Stderr, "Running %s...\n", ec.Name)
		results[i] = runEval(client, ec)
		mark := "✗"
		if results[i].Passed {
			mark = "✓"
		}
		fmt.Fprintf(os.Stderr, "  %s (attempts: %d)\n", mark, results[i].Attempts)
	}

	printSummary(model, results)
}

func runEval(client *Client, ec Case) evalResult {
	const maxAttempts = 3
	const maxIterations = 6

	systemPrompt := "You are solving a programming task using the execute-starlark tool. " +
		"Use the tool to write and run a Starlark program that produces the requested output. " +
		"Do not explain your work — just call the tool." +
		"\n\nThe following documentation describes the built-in functions available:\n\n" +
		server.BuiltinsDocumentation

	toolDef := ToolDef{
		Name:        "execute-starlark",
		Description: server.ExecuteStarlarkDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"program": map[string]any{
					"type": "string",
				},
				"timeout_secs": map[string]any{
					"type": "number",
				},
			},
			"required": []string{"program", "timeout_secs"},
		},
	}

	messages := []Message{
		{
			Role:    "user",
			Content: []map[string]any{TextBlock(ec.Prompt)},
		},
	}

	result := evalResult{
		Case: ec,
	}

	for iter := 0; iter < maxIterations; iter++ {
		if result.Attempts >= maxAttempts {
			break
		}

		req := &Request{
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     []ToolDef{toolDef},
		}

		resp, err := client.SendRequest(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "API request failed for %s: %v\n", ec.Name, err)
			break
		}

		result.TokensIn += resp.Usage.InputTokens
		result.TokensOut += resp.Usage.OutputTokens

		// Append assistant message to conversation.
		messages = append(messages, ResponseToMessage(resp))

		// Find tool_use block.
		var toolUse *ResponseContentBlock
		for idx := range resp.Content {
			if resp.Content[idx].Type == "tool_use" {
				toolUse = &resp.Content[idx]
				break
			}
		}

		if toolUse == nil {
			// No tool use, nudge the model.
			messages = append(messages, Message{
				Role:    "user",
				Content: []map[string]any{TextBlock("Please use the execute-starlark tool.")},
			})
			continue
		}

		// Parse tool input.
		var input ToolInput
		if err := json.Unmarshal(toolUse.Input, &input); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse tool input for %s: %v\n", ec.Name, err)
			break
		}

		// Execute with a fixed 30s timeout.
		execCtx, execCancel := context.WithTimeout(context.Background(), 30*time.Second)
		output, execErr := server.ExecuteStarlark(execCtx, input.Program)
		execCancel()

		result.Attempts++

		if execErr != nil {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %v", execErr))
			messages = append(messages, Message{
				Role: "user",
				Content: []map[string]any{
					ToolResultBlock(toolUse.ID, execErr.Error(), true),
				},
			})
			continue
		}

		result.Outputs = append(result.Outputs, output)

		if ec.Judge(output) {
			result.Passed = true
			result.Score = 1.0 / math.Pow(2, float64(result.Attempts-1))
			return result
		}

		// Judge failed. If we still have attempts, send tool result + nudge.
		if result.Attempts < maxAttempts {
			messages = append(messages, Message{
				Role: "user",
				Content: []map[string]any{
					ToolResultBlock(toolUse.ID, output, false),
					TextBlock("The output did not match the expected result. Please try again with a corrected program."),
				},
			})
		}
	}

	result.Passed = false
	result.Score = 0.0
	return result
}

func printSummary(model string, results []evalResult) {
	tierNames := map[int]string{
		1: "BASICS",
		2: "SIMPLE ALGORITHMS",
		3: "INTERMEDIATE",
		4: "HARD",
	}

	fmt.Printf("\n%s\n", strings.Repeat("═", 62))
	fmt.Printf("EVAL RESULTS — model: %s\n", model)
	fmt.Println(strings.Repeat("═", 62))

	totalPassed := 0
	totalCases := 0
	totalScore := 0.0
	totalTokensIn := 0
	totalTokensOut := 0

	for tier := 1; tier <= 4; tier++ {
		var tierResults []evalResult
		for _, r := range results {
			if r.Case.Tier == tier {
				tierResults = append(tierResults, r)
			}
		}
		if len(tierResults) == 0 {
			continue
		}

		fmt.Printf("\nTIER %d: %s\n", tier, tierNames[tier])

		tierPassed := 0
		tierTotal := len(tierResults)
		tierScore := 0.0

		for _, r := range tierResults {
			mark := "✗"
			if r.Passed {
				mark = "✓"
				tierPassed++
			}
			name := r.Case.Name
			padding := 35 - len(name)
			if padding < 1 {
				padding = 1
			}
			fmt.Printf("  %s %s%s attempts: %d  score: %.2f\n",
				mark, name, strings.Repeat(" ", padding), r.Attempts, r.Score)
			tierScore += r.Score
			totalTokensIn += r.TokensIn
			totalTokensOut += r.TokensOut
		}

		fmt.Printf("  Tier score: %.2f (%d/%d passed)\n",
			tierScore/float64(tierTotal), tierPassed, tierTotal)

		totalPassed += tierPassed
		totalCases += tierTotal
		totalScore += tierScore
	}

	fmt.Printf("\n%s\n", strings.Repeat("─", 62))
	fmt.Printf("OVERALL: %.2f (%d/%d passed)  tokens: %d in, %d out\n",
		totalScore/float64(totalCases), totalPassed, totalCases, totalTokensIn, totalTokensOut)
	fmt.Println(strings.Repeat("─", 62))
}
