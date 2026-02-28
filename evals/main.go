//go:build eval

// Command eval runs the LLM eval harness against the Starlark MCP server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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
	Duration  time.Duration
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

	disp := newDisplay(Cases)
	results := make([]evalResult, len(Cases))
	var wg sync.WaitGroup
	for i, ec := range Cases {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each eval gets its own MCP session for isolation.
			ctx := context.Background()
			t1, t2 := mcp.NewInMemoryTransports()
			srv := server.New()
			mcpClient := mcp.NewClient(&mcp.Implementation{Name: "eval-client"}, nil)

			if _, err := srv.Connect(ctx, t1, nil); err != nil {
				disp.finish(i, false, 0, 0)
				return
			}
			session, err := mcpClient.Connect(ctx, t2, nil)
			if err != nil {
				disp.finish(i, false, 0, 0)
				return
			}
			defer session.Close()

			toolDefs, err := mcpToolDefs(ctx, session)
			if err != nil {
				disp.finish(i, false, 0, 0)
				return
			}

			results[i] = runEval(llm, session, toolDefs, ec)
			results[i].Duration = time.Since(disp.startTimes[i])
			disp.finish(i, results[i].Passed, results[i].Attempts, results[i].Duration)
		}()
	}
	wg.Wait()
	disp.stop()

	printSummary(model, results)
}

// display manages live terminal output for eval progress.
type display struct {
	mu         sync.Mutex
	cases      []Case
	sorted     []int // indices into cases, sorted lexicographically
	startTimes []time.Time
	done       []bool
	passed     []bool
	attempts   []int
	durations  []time.Duration
	stopCh     chan struct{}
}

func newDisplay(cases []Case) *display {
	now := time.Now()
	sorted := make([]int, len(cases))
	for i := range sorted {
		sorted[i] = i
	}
	sort.Slice(sorted, func(a, b int) bool {
		return cases[sorted[a]].Name < cases[sorted[b]].Name
	})
	d := &display{
		cases:      cases,
		sorted:     sorted,
		startTimes: make([]time.Time, len(cases)),
		done:       make([]bool, len(cases)),
		passed:     make([]bool, len(cases)),
		attempts:   make([]int, len(cases)),
		durations:  make([]time.Duration, len(cases)),
		stopCh:     make(chan struct{}),
	}
	for i := range cases {
		d.startTimes[i] = now
	}
	// Print initial lines.
	for range cases {
		fmt.Fprint(os.Stderr, "\n")
	}
	d.render()
	go d.loop()
	return d
}

func (d *display) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.render()
		}
	}
}

func (d *display) finish(i int, passed bool, attempts int, dur time.Duration) {
	d.mu.Lock()
	d.done[i] = true
	d.passed[i] = passed
	d.attempts[i] = attempts
	d.durations[i] = dur
	d.mu.Unlock()
	d.render()
}

func (d *display) stop() {
	close(d.stopCh)
	d.render()
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (d *display) render() {
	d.mu.Lock()
	defer d.mu.Unlock()

	n := len(d.cases)
	// Move cursor up n lines.
	fmt.Fprintf(os.Stderr, "\033[%dA", n)

	now := time.Now()
	frame := int(now.UnixMilli()/80) % len(spinnerFrames)
	for _, i := range d.sorted {
		c := d.cases[i]
		// Clear line and write status.
		fmt.Fprintf(os.Stderr, "\033[2K")
		if d.done[i] {
			if d.passed[i] {
				fmt.Fprintf(os.Stderr, "  %s✔ %s%s %s(%.1fs, %d attempts)%s\n",
					colorGreen, c.Name, colorReset, colorDim, d.durations[i].Seconds(), d.attempts[i], colorReset)
			} else {
				fmt.Fprintf(os.Stderr, "  %s✘ %s%s %s(%.1fs, %d attempts)%s\n",
					colorRed, c.Name, colorReset, colorDim, d.durations[i].Seconds(), d.attempts[i], colorReset)
			}
		} else {
			elapsed := now.Sub(d.startTimes[i])
			fmt.Fprintf(os.Stderr, "  %s%s %s%s %s[T%d] (%.1fs)%s\n",
				colorYellow, spinnerFrames[frame], c.Name, colorReset, colorDim, c.Tier, elapsed.Seconds(), colorReset)
		}
	}
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

	fmt.Printf("\n%s%s%s\n", colorCyan, strings.Repeat("═", 62), colorReset)
	fmt.Printf("%s%sEVAL RESULTS — model: %s%s\n", colorBold, colorCyan, model, colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", 62), colorReset)

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

		fmt.Printf("\n%s%sTIER %d: %s%s\n", colorBold, colorCyan, tier, tierNames[tier], colorReset)

		tierPassed := 0
		tierTotal := len(tierResults)
		tierScore := 0.0

		for _, r := range tierResults {
			var mark, color string
			if r.Passed {
				mark = "✔"
				color = colorGreen
				tierPassed++
			} else {
				mark = "✘"
				color = colorRed
			}
			name := r.Case.Name
			padding := 35 - len(name)
			if padding < 1 {
				padding = 1
			}
			fmt.Printf("  %s%s%s %s%s%sattempts: %d  score: %.2f  %.1fs%s\n",
				color, mark, colorReset, name, strings.Repeat(" ", padding), colorDim, r.Attempts, r.Score, r.Duration.Seconds(), colorReset)
			tierScore += r.Score
			totalTokensIn += r.TokensIn
			totalTokensOut += r.TokensOut
		}

		fmt.Printf("  %sTier score: %.2f (%d/%d passed)%s\n",
			colorDim, tierScore/float64(tierTotal), tierPassed, tierTotal, colorReset)

		totalPassed += tierPassed
		totalCases += tierTotal
		totalScore += tierScore
	}

	fmt.Printf("\n%s%s%s\n", colorCyan, strings.Repeat("─", 62), colorReset)
	fmt.Printf("%s%sOVERALL: %.2f (%d/%d passed)  tokens: %d in, %d out%s\n",
		colorBold, colorCyan, totalScore/float64(totalCases), totalPassed, totalCases, totalTokensIn, totalTokensOut, colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("─", 62), colorReset)
}
