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

Starlark is a Python-like language with important restrictions and syntax differences.

KEY SYNTAX DIFFERENCES FROM PYTHON:
- All top-level code must be in functions (no bare loops/conditionals)
- Operator chaining requires parentheses: use (a <= b) and (b < c) not a <= b < c  
- No f-strings or % formatting - use string concatenation with str()
- No tuple unpacking in assignments beyond simple cases
- More restrictive about operator precedence

STARLARK RESTRICTIONS:
- No file I/O, network access, or system calls
- No imports except built-in functions
- No while loops (use for loops with range)
- No classes or complex OOP features
- Deterministic execution only

EXAMPLE PROGRAM STRUCTURE:

def my_function():
  result = []
  for i in range(10):
    result.append(str(i))
  return result

def main():
  data = my_function()
  for item in data:
    print(item)

main()  # Must call explicitly

COMMON PATTERNS:
- String building: use concatenation like s = s + "text"
- Avoid complex expressions: break into multiple lines
- Use explicit str() conversion for print statements
- Put all execution logic in functions

REFERENCE:
See https://raw.githubusercontent.com/google/starlark-go/bf296ed553ea1715656054a7f64ac6a6dd161360/doc/spec.md
for the Starlark language specification.
`

var ExecuteStarlarkTool = &mcp.Tool{
	Name:        "execute-starlark",
	Description: ExecuteStarlarkDescription,
}

const ServerName = "starlark-mcp"

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
