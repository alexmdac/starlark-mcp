package main

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed builtins.md
var builtinsDocumentation string

var embeddedResources = map[string]string{
	"builtins": builtinsDocumentation,
}

func embeddedResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, err := url.Parse(req.Params.URI)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "starlark" {
		return nil, fmt.Errorf("wrong scheme: %q", u.Scheme)
	}
	key := u.Host
	text, ok := embeddedResources[key]
	if !ok {
		return nil, fmt.Errorf("no embedded resource named %q", key)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "text/plain", Text: text},
		},
	}, nil
}

func addEmbeddedResources(server *mcp.Server) {
	for resourceName := range embeddedResources {
		server.AddResource(&mcp.Resource{
			Name:     resourceName,
			MIMEType: "text/plain",
			URI:      fmt.Sprintf("starlark://%s", resourceName),
		}, embeddedResource)
	}
}
