//go:build eval

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

type evalResult struct {
	Case      evalCase
	Passed    bool
	Attempts  int
	Score     float64
	Outputs   []string // starlark output from each attempt
	TokensIn  int
	TokensOut int
}

func TestEval(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping eval")
	}

	model := os.Getenv("EVAL_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	client := newLLMClient(apiKey, model, baseURL)

	results := make([]evalResult, len(evalCases))

	for i, ec := range evalCases {
		ec := ec
		i := i
		t.Run(ec.Name, func(t *testing.T) {
			result := runEval(t, client, ec)
			results[i] = result
			if !result.Passed {
				t.Errorf("eval case %q failed after %d attempts", ec.Name, result.Attempts)
				for j, out := range result.Outputs {
					t.Logf("  attempt %d output: %q", j+1, out)
				}
			}
		})
	}

	// Print summary table.
	printSummary(t, model, results)
}

func runEval(t *testing.T, client *llmClient, ec evalCase) evalResult {
	t.Helper()

	const maxAttempts = 3
	const maxIterations = 6

	systemPrompt := "You are solving a programming task using the execute-starlark tool. " +
		"Use the tool to write and run a Starlark program that produces the requested output. " +
		"Do not explain your work \u2014 just call the tool." +
		"\n\nThe following documentation describes the built-in functions available:\n\n" +
		builtinsDocumentation

	toolDef := apiToolDef{
		Name:        "execute-starlark",
		Description: executeStarlarkDescription,
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

	messages := []apiMessage{
		{
			Role:    "user",
			Content: []map[string]any{textBlock(ec.Prompt)},
		},
	}

	result := evalResult{
		Case: ec,
	}

	for iter := 0; iter < maxIterations; iter++ {
		if result.Attempts >= maxAttempts {
			break
		}

		req := &apiRequest{
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     []apiToolDef{toolDef},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		resp, err := client.sendRequest(ctx, req)
		cancel()
		if err != nil {
			t.Fatalf("API request failed: %v", err)
		}

		result.TokensIn += resp.Usage.InputTokens
		result.TokensOut += resp.Usage.OutputTokens

		// Append assistant message to conversation.
		messages = append(messages, responseToMessage(resp))

		// Find tool_use block.
		var toolUse *apiResponseContentBlock
		for idx := range resp.Content {
			if resp.Content[idx].Type == "tool_use" {
				toolUse = &resp.Content[idx]
				break
			}
		}

		if toolUse == nil {
			// No tool use, nudge the model.
			messages = append(messages, apiMessage{
				Role:    "user",
				Content: []map[string]any{textBlock("Please use the execute-starlark tool.")},
			})
			continue
		}

		// Parse tool input.
		var input toolInput
		if err := json.Unmarshal(toolUse.Input, &input); err != nil {
			t.Fatalf("failed to parse tool input: %v", err)
		}

		// Execute with a fixed 30s timeout.
		execCtx, execCancel := context.WithTimeout(context.Background(), 30*time.Second)
		output, execErr := executeStarlark(execCtx, input.Program)
		execCancel()

		result.Attempts++

		if execErr != nil {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %v", execErr))
			// Append error tool result.
			messages = append(messages, apiMessage{
				Role: "user",
				Content: []map[string]any{
					toolResultBlock(toolUse.ID, execErr.Error(), true),
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
			messages = append(messages, apiMessage{
				Role: "user",
				Content: []map[string]any{
					toolResultBlock(toolUse.ID, output, false),
					textBlock("The output did not match the expected result. Please try again with a corrected program."),
				},
			})
		}
	}

	result.Passed = false
	result.Score = 0.0
	return result
}

func printSummary(t *testing.T, model string, results []evalResult) {
	t.Helper()

	var sb strings.Builder

	sb.WriteString("\n\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\n")
	sb.WriteString(fmt.Sprintf("EVAL RESULTS \u2014 model: %s\n", model))
	sb.WriteString("\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\n")

	tierNames := map[int]string{
		1: "BASICS",
		2: "SIMPLE ALGORITHMS",
		3: "INTERMEDIATE",
		4: "HARD",
	}

	totalPassed := 0
	totalCases := 0
	totalScore := 0.0
	totalTokensIn := 0
	totalTokensOut := 0

	for tier := 1; tier <= 4; tier++ {
		// Gather results for this tier.
		var tierResults []evalResult
		for _, r := range results {
			if r.Case.Tier == tier {
				tierResults = append(tierResults, r)
			}
		}
		if len(tierResults) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\nTIER %d: %s\n", tier, tierNames[tier]))

		tierPassed := 0
		tierTotal := len(tierResults)
		tierScore := 0.0

		for _, r := range tierResults {
			mark := "\u2717"
			if r.Passed {
				mark = "\u2713"
				tierPassed++
			}
			// Pad name to 35 chars for alignment.
			name := r.Case.Name
			padding := 35 - len(name)
			if padding < 1 {
				padding = 1
			}
			sb.WriteString(fmt.Sprintf("  %s %s%s attempts: %d  score: %.2f\n",
				mark, name, strings.Repeat(" ", padding), r.Attempts, r.Score))
			tierScore += r.Score
			totalTokensIn += r.TokensIn
			totalTokensOut += r.TokensOut
		}

		sb.WriteString(fmt.Sprintf("  Tier score: %.2f (%d/%d passed)\n",
			tierScore/float64(tierTotal), tierPassed, tierTotal))

		totalPassed += tierPassed
		totalCases += tierTotal
		totalScore += tierScore
	}

	sb.WriteString("\n\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n")
	sb.WriteString(fmt.Sprintf("OVERALL: %.2f (%d/%d passed)  tokens: %d in, %d out\n",
		totalScore/float64(totalCases), totalPassed, totalCases, totalTokensIn, totalTokensOut))
	sb.WriteString("\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n")

	t.Logf("%s", sb.String())
}
