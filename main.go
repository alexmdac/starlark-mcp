package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "starlark-mcp"}, nil)
	addEmbeddedResources(server)
	addExecuteStarlarkTool(server)
	return server
}

func main() {
	server := newMCPServer()
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
