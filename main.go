package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

//go:embed execute_starlark_description.md
var executeStarlarkDescription string

const (
	serverName          = "starlark-mcp"
	executeStarlarkName = "execute-starlark"
)

func runMCPServer(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName}, nil)
	executeStarlarkTool := &mcp.Tool{
		Name:        executeStarlarkName,
		Description: executeStarlarkDescription,
	}
	mcp.AddTool(server, executeStarlarkTool, executeStarlark)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func main() {
	ctx := context.Background()
	if err := runMCPServer(ctx); err != nil {
		log.Fatal(err)
	}
}
