package server

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New creates a configured MCP server with all tools and resources registered.
func New() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "starlark-mcp"}, nil)
	addEmbeddedResources(s)
	addExecuteStarlarkTool(s)
	return s
}
