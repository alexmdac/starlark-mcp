package main

import (
	"context"
	"log"

	"github.com/alexmdac/starlark-mcp/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	s := server.New()
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
