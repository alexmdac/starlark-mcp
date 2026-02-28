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

func main() {
	runsFlag := flag.Int("runs", 5, "number of independent runs per eval case")
	flag.Parse()
	numRuns := *runsFlag
	if numRuns < 1 {
		numRuns = 1
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = "unspecified"
	}

	model := os.Getenv("EVAL_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "http://169.254.169.254/gateway/llm/anthropic"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	llm := newClient(apiKey, model, baseURL)

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
				res := runSingleEval(ctx, llm, ec)
				res.Duration = time.Since(start)
				allResults[i].Runs[r] = res
				disp.finishRun(i, res.Passed, res.Duration)
			}()
		}
	}
	wg.Wait()
	disp.stop()

	printSummary(model, numRuns, allResults)
}

// runSingleEval sets up an isolated MCP session and runs a single eval case.
func runSingleEval(ctx context.Context, llm *client, ec evalCase) evalResult {
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

	return runEval(ctx, llm, session, toolDefs, ec)
}

// mcpToolDefs calls ListTools on the MCP session and converts the results
// into the ToolDef format expected by the Anthropic Messages API.
func mcpToolDefs(ctx context.Context, session *mcp.ClientSession) ([]toolDef, error) {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	defs := make([]toolDef, len(res.Tools))
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
		defs[i] = toolDef{
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

func runEval(ctx context.Context, llm *client, session *mcp.ClientSession, toolDefs []toolDef, ec evalCase) evalResult {
	const maxAttempts = 3
	const maxIterations = 6

	const systemPrompt = "You have access to tools. Use them to solve the task. " +
		"Do not explain your work — just call the appropriate tool."

	messages := []message{
		{
			Role:    "user",
			Content: []map[string]any{textBlock(ec.prompt)},
		},
	}

	result := evalResult{ec: ec}

	for iter := 0; iter < maxIterations; iter++ {
		if result.Attempts >= maxAttempts {
			break
		}

		req := &request{
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     toolDefs,
		}

		llmStart := time.Now()
		resp, err := llm.sendRequest(ctx, req)
		result.LLMTime += time.Since(llmStart)
		if err != nil {
			break
		}

		result.TokensIn += resp.Usage.InputTokens
		result.TokensOut += resp.Usage.OutputTokens

		messages = append(messages, responseToMessage(resp))

		// Find tool_use block.
		var toolUse *responseContentBlock
		for idx := range resp.Content {
			if resp.Content[idx].Type == "tool_use" {
				toolUse = &resp.Content[idx]
				break
			}
		}

		if toolUse == nil {
			break
		}

		// Call the tool via MCP.
		toolStart := time.Now()
		output, toolIsError, callErr := callMCPTool(
			ctx, session, toolUse.Name, toolUse.Input,
		)
		result.StarlarkTime += time.Since(toolStart)

		result.Attempts++

		if callErr != nil {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %v", callErr))
			messages = append(messages, message{
				Role: "user",
				Content: []map[string]any{
					toolResultBlock(toolUse.ID, callErr.Error(), true),
				},
			})
			continue
		}

		if toolIsError {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %s", output))
			messages = append(messages, message{
				Role: "user",
				Content: []map[string]any{
					toolResultBlock(toolUse.ID, output, true),
				},
			})
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
			messages = append(messages, message{
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
