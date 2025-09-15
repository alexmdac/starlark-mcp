package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ExecuteStarlarkParams struct {
	Program string `json:"program" jsonschema:"a valid Starlark program"`
}

func ExecuteStarlark(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args ExecuteStarlarkParams,
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

const ExecuteStarlarkDescription = `Executes Starlark programs.

See https://raw.githubusercontent.com/google/starlark-go/bf296ed553ea1715656054a7f64ac6a6dd161360/doc/spec.md
for the Starlark language specification.

Programs can emit output using the Starlark "print" statement. The "print" statement
outputs the given string with a newline added. The MCP server returns the aggregated
output.`

var ExecuteStarlarkTool = &mcp.Tool{
	Name:        "ExecuteStarlark",
	Description: ExecuteStarlarkDescription,
}

const ServerName = "StarlarkMCP"

func RunMCPServer(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{Name: ServerName}, nil)
	mcp.AddTool(server, ExecuteStarlarkTool, ExecuteStarlark)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func main() {
	ctx := context.Background()
	if err := RunMCPServer(ctx); err != nil {
		log.Fatal(err)
	}
}
