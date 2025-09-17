package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

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
	Program string `json:"program" jsonschema:"a valid Starlark program"`
}

func executeStarlark(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args executeStarlarkParams,
) (*mcp.CallToolResult, any, error) {
	var buf bytes.Buffer
	thread := &starlark.Thread{
		Print: func(thread *starlark.Thread, msg string) {
			buf.WriteString(msg) // This panics on OOM, never returns a non-nil error.
			buf.WriteRune('\n')
		},
		Load: loadBuiltinModule,
	}
	_, err := starlark.ExecFileOptions(
		syntax.LegacyFileOptions(),
		thread,
		"LLM supplied program",
		args.Program,
		nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute program: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: buf.String()},
		},
	}, nil, nil
}
