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

//go:embed execute_starlark_description.md
var executeStarlarkDescription string

func addExecuteStarlarkTool(server *mcp.Server) {
	executeStarlarkTool := &mcp.Tool{
		Name:        "execute-starlark",
		Description: executeStarlarkDescription,
	}
	mcp.AddTool(server, executeStarlarkTool, executeStarlark)
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

func executeStarlark(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args executeStarlarkParams,
) (*mcp.CallToolResult, any, error) {
	if err := args.validate(); err != nil {
		return nil, nil, err
	}

	ctx, done := context.WithTimeout(ctx, args.timeout())
	defer done()

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
		args.Program,
		predeclared())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute program: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: buf.String()},
		},
	}, nil, nil
}
