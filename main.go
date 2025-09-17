package main

import (
	"context"
	_ "embed"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runMCPServer(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "starlark-mcp"}, nil)
	addEmbeddedResources(server)
	addExecuteStarlarkTool(server)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func main() {
	ctx := context.Background()
	if err := runMCPServer(ctx); err != nil {
		log.Fatal(err)
	}
}
