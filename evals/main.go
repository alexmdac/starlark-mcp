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
	"sync"

	"github.com/alexmdac/starlark-mcp/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		apiKey = "unspecified"
	}

	model := os.Getenv("EVAL_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	llm := NewClient(apiKey, model, baseURL)

	results := make([]evalResult, len(Cases))
	var wg sync.WaitGroup
	for i, ec := range Cases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Fprintf(os.Stderr, "Running %s...\n", ec.Name)

			// Each eval gets its own MCP session for isolation.
			ctx := context.Background()
			t1, t2 := mcp.NewInMemoryTransports()
			srv := server.New()
			mcpClient := mcp.NewClient(&mcp.Implementation{Name: "eval-client"}, nil)

			if _, err := srv.Connect(ctx, t1, nil); err != nil {
				fmt.Fprintf(os.Stderr, "  %s: failed to connect MCP server: %v\n", ec.Name, err)
				return
			}
			session, err := mcpClient.Connect(ctx, t2, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s: failed to connect MCP client: %v\n", ec.Name, err)
				return
			}
			defer session.Close()

			toolDefs, err := mcpToolDefs(ctx, session)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s: failed to list tools: %v\n", ec.Name, err)
				return
			}

			results[i] = runEval(llm, session, toolDefs, ec)
			mark := "✗"
			if results[i].Passed {
				mark = "✓"
			}
			fmt.Fprintf(os.Stderr, "  %s %s (attempts: %d)\n", mark, ec.Name, results[i].Attempts)
		}()
	}
	wg.Wait()

	printSummary(model, results)
}

// mcpToolDefs calls ListTools on the MCP session and converts the results
// into the ToolDef format expected by the Anthropic Messages API.
func mcpToolDefs(ctx context.Context, session *mcp.ClientSession) ([]ToolDef, error) {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	defs := make([]ToolDef, len(res.Tools))
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
		defs[i] = ToolDef{
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

func runEval(llm *Client, session *mcp.ClientSession, toolDefs []ToolDef, ec Case) evalResult {
	const maxAttempts = 3
	const maxIterations = 6

	const systemPrompt = "You have access to tools. Use them to solve the task. " +
		"Do not explain your work — just call the appropriate tool."

	messages := []Message{
		{
			Role:    "user",
			Content: []map[string]any{TextBlock(ec.Prompt)},
		},
	}

	result := evalResult{Case: ec}

	for iter := 0; iter < maxIterations; iter++ {
		if result.Attempts >= maxAttempts {
			break
		}

		req := &Request{
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     toolDefs,
		}

		resp, err := llm.SendRequest(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "API request failed for %s: %v\n", ec.Name, err)
			break
		}

		result.TokensIn += resp.Usage.InputTokens
		result.TokensOut += resp.Usage.OutputTokens

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
			break
		}

		// Call the tool via MCP.
		output, toolIsError, callErr := callMCPTool(
			context.Background(), session, toolUse.Name, toolUse.Input,
		)

		result.Attempts++

		if callErr != nil {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %v", callErr))
			messages = append(messages, Message{
				Role: "user",
				Content: []map[string]any{
					ToolResultBlock(toolUse.ID, callErr.Error(), true),
				},
			})
			continue
		}

		if toolIsError {
			result.Outputs = append(result.Outputs, fmt.Sprintf("ERROR: %s", output))
			messages = append(messages, Message{
				Role: "user",
				Content: []map[string]any{
					ToolResultBlock(toolUse.ID, output, true),
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
