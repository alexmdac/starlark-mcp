package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

const executeStarlarkName = "execute-starlark"

//go:embed execute_starlark_description.md
var executeStarlarkDescription string

func addExecuteStarlarkTool(server *mcp.Server) {
	tool := &mcp.Tool{
		Name:        executeStarlarkName,
		Description: executeStarlarkDescription,
	}
	mcp.AddTool(server, tool, handleExecuteStarlarkTool)
}

type executeStarlarkParams struct {
	Program     string  `json:"program" jsonschema:"a valid Starlark program"`
	TimeoutSecs float32 `json:"timeout_secs" jsonschema:"execution timeout in seconds"`
}

func (p executeStarlarkParams) validate() error {
	if p.TimeoutSecs <= 0.0 {
		return fmt.Errorf("invalid timeout: %f", p.TimeoutSecs)
	}
	return nil
}

func (p executeStarlarkParams) timeout() time.Duration {
	return time.Duration(p.TimeoutSecs * float32(time.Second))
}

func handleExecuteStarlarkTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args executeStarlarkParams,
) (*mcp.CallToolResult, any, error) {
	if err := args.validate(); err != nil {
		return nil, nil, err
	}

	ctx, done := context.WithTimeout(ctx, args.timeout())
	defer done()

	output, err := executeStarlark(ctx, args.Program)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

// executeStarlark executes the given Starlark program and returns its output.
// The program generates output using the "print" builtin function.
func executeStarlark(ctx context.Context, program string) (string, error) {
	var buf bytes.Buffer
	thread := &starlark.Thread{
		Print: func(thread *starlark.Thread, msg string) {
			buf.WriteString(msg) // This panics on OOM, never returns a non-nil error.
			buf.WriteRune('\n')
		},
		Load: loadBuiltinModule,
	}
	context.AfterFunc(ctx, func() {
		reason := ""
		if err := ctx.Err(); err != nil {
			reason = err.Error()
		}
		thread.Cancel(reason)
	})

	_, err := starlark.ExecFileOptions(
		syntax.LegacyFileOptions(),
		thread,
		"LLM supplied program",
		program,
		predeclared())
	if err != nil {
		return "", fmt.Errorf("failed to execute program: %v", err)
	}
	return buf.String(), nil
}
