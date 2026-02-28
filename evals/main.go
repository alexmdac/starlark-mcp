//go:build eval

// Command eval runs the LLM eval harness against the Starlark MCP server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/alexmdac/starlark-mcp/internal/llm"
	"github.com/alexmdac/starlark-mcp/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type evalResult struct {
	ec           evalCase
	Passed       bool
	Attempts     int
	Score        float64
	Outputs      []string // starlark output from each attempt
	TokensIn     int
	TokensOut    int
	Duration     time.Duration
	LLMTime      time.Duration
	StarlarkTime time.Duration
}

// caseResults holds all runs for a single eval case.
type caseResults struct {
	ec   evalCase
	Runs []evalResult
}

const (
	defaultAnthropicURL = "http://169.254.169.254/gateway/llm/anthropic"
	defaultOpenAIURL    = "http://169.254.169.254/gateway/llm/openai"
)

func main() {
	runsFlag := flag.Int("runs", 5, "number of independent runs per eval case")
	llmFlag := flag.String("llm", "anthropic:claude-sonnet-4-6", "provider:model (e.g. \"anthropic:claude-haiku-4-5\")")
	llmURLFlag := flag.String("llm-url", "", "base URL for the LLM API (overrides provider default)")
	flag.Parse()
	numRuns := *runsFlag
	if numRuns < 1 {
		numRuns = 1
	}

	providerName, model, err := llm.ParseModel(*llmFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	baseURL := *llmURLFlag

	var client llm.Client
	switch providerName {
	case "anthropic":
		if baseURL == "" {
			baseURL = defaultAnthropicURL
		}
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = "unspecified"
		}
		client = llm.NewAnthropic(apiKey, model, baseURL)
	case "openai":
		if baseURL == "" {
			baseURL = defaultOpenAIURL
		}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = "unspecified"
		}
		client = llm.NewOpenAI(apiKey, model, baseURL)
	default:
		fmt.Fprintf(os.Stderr, "unknown provider: %q (supported: anthropic, openai)\n", providerName)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Limit concurrency to avoid API rate limits.
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)

	disp := newDisplay(cases, numRuns)
	allResults := make([]caseResults, len(cases))
	for i := range allResults {
		allResults[i] = caseResults{
			ec:   cases[i],
			Runs: make([]evalResult, numRuns),
		}
	}

	var wg sync.WaitGroup
	for i, ec := range cases {
		for r := range numRuns {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				start := time.Now()
				res := runSingleEval(ctx, client, ec)
				res.Duration = time.Since(start)
				allResults[i].Runs[r] = res
				disp.finishRun(i, res.Passed, res.Duration)
			}()
		}
	}
	wg.Wait()
	disp.stop()

	printSummary(*llmFlag, numRuns, allResults)
}

// runSingleEval sets up an isolated MCP session and runs a single eval case.
func runSingleEval(ctx context.Context, client llm.Client, ec evalCase) evalResult {
	t1, t2 := mcp.NewInMemoryTransports()
	srv := server.New()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "eval-client"}, nil)

	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		return evalResult{ec: ec}
	}
	session, err := mcpClient.Connect(ctx, t2, nil)
	if err != nil {
		return evalResult{ec: ec}
	}
	defer session.Close()

	toolDefs, err := mcpToolDefs(ctx, session)
	if err != nil {
		return evalResult{ec: ec}
	}

	return runEval(ctx, client, session, toolDefs, ec)
}

// mcpToolDefs calls ListTools on the MCP session and converts the results
// into the llm.ToolDef format.
func mcpToolDefs(ctx context.Context, session *mcp.ClientSession) ([]llm.ToolDef, error) {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	defs := make([]llm.ToolDef, len(res.Tools))
	for i, tool := range res.Tools {
		// Convert the JSON Schema to map[string]any via JSON round-trip.
		var schema map[string]any
		if tool.InputSchema != nil {
			b, err := json.Marshal(tool.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("marshal schema for %s: %w", tool.Name, err)
			}
			if err := json.Unmarshal(b, &schema); err != nil {
				return nil, fmt.Errorf("unmarshal schema for %s: %w", tool.Name, err)
			}
		}
		defs[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		}
	}
	return defs, nil
}

// callMCPTool invokes a tool on the MCP server and returns the text output.
func callMCPTool(ctx context.Context, session *mcp.ClientSession, name string, rawInput json.RawMessage) (output string, isError bool, err error) {
	var args map[string]any
	if err := json.Unmarshal(rawInput, &args); err != nil {
		return "", false, fmt.Errorf("unmarshal tool input: %w", err)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", false, fmt.Errorf("CallTool %s: %w", name, err)
	}

	// Extract text content from the result.
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String(), res.IsError, nil
}

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

func runEval(ctx context.Context, client llm.Client, session *mcp.ClientSession, toolDefs []llm.ToolDef, ec evalCase) evalResult {
	const maxAttempts = 3
	const maxIterations = 6

	const systemPrompt = "You have access to tools. Use them to solve the task. " +
		"Do not explain your work — just call the appropriate tool."

	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			Text: ec.prompt,
		},
	}

	result := evalResult{ec: ec}

	for iter := 0; iter < maxIterations; iter++ {
		if result.Attempts >= maxAttempts {
			break
		}

		params := &llm.MessageParams{
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     toolDefs,
		}

		llmStart := time.Now()
		resp, err := client.SendMessage(ctx, params)
		result.LLMTime += time.Since(llmStart)
		if err != nil {
			break
		}

		result.TokensIn += resp.Usage.InputTokens
		result.TokensOut += resp.Usage.OutputTokens

		messages = append(messages, responseToHistory(resp))

		// Find the first tool call.
		if len(resp.ToolCalls) == 0 {
			break
		}
		toolCall := resp.ToolCalls[0]

		// Call the tool via MCP.
		toolStart := time.Now()
		output, toolIsError, callErr := callMCPTool(
			ctx, session, toolCall.Name, toolCall.Input,
		)
		result.StarlarkTime += time.Since(toolStart)

		result.Attempts++

		if callErr != nil {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %v", callErr))
			messages = append(messages, toolResultMessage(toolCall.ID, callErr.Error(), true))
			continue
		}

		if toolIsError {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %s", output))
			messages = append(messages, toolResultMessage(toolCall.ID, output, true))
			continue
		}

		result.Outputs = append(result.Outputs, output)

		if ec.judge(output) {
			result.Passed = true
			result.Score = 1.0 / math.Pow(2, float64(result.Attempts-1))
			return result
		}

		// Judge failed. If we still have attempts, send tool result + nudge.
		if result.Attempts < maxAttempts {
			messages = append(messages, toolResultWithNudge(
				toolCall.ID, output,
				"The output did not match the expected result. Please try again with a corrected program.",
			))
		}
	}

	result.Passed = false
	result.Score = 0.0
	return result
}
