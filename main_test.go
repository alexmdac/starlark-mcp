package main

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func startTestServer(t *testing.T) *mcp.ClientSession {
	t.Helper()

	t1, t2 := mcp.NewInMemoryTransports()
	server := newMCPServer()
	client := mcp.NewClient(&mcp.Implementation{Name: "test client"}, nil)

	serverSession, err := server.Connect(t.Context(), t1, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}
	clientSession, err := client.Connect(t.Context(), t2, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	t.Cleanup(func() {
		if err := clientSession.Close(); err != nil {
			t.Fatalf("Failed to close client session: %v", err)
		}
		if err := serverSession.Wait(); err != nil {
			t.Fatalf("Server session failed: %v", err)
		}
	})

	return clientSession
}

func expectCallToolSuccess(t *testing.T, client *mcp.ClientSession, params *mcp.CallToolParams) string {
	t.Helper()
	res := callTool(t, client, params)
	if res.IsError {
		t.Fatalf("Expected tool call to succeed, but it failed. Full result: %#v", res)
	}
	return expectTextContent(t, res)
}

func expectCallToolError(t *testing.T, client *mcp.ClientSession, params *mcp.CallToolParams) string {
	t.Helper()
	res := callTool(t, client, params)
	if !res.IsError {
		t.Fatal("expected an error, but got none")
	}
	return expectTextContent(t, res)
}

func callTool(t *testing.T, client *mcp.ClientSession, params *mcp.CallToolParams) *mcp.CallToolResult {
	t.Helper()
	res, err := client.CallTool(t.Context(), params)
	if err != nil {
		t.Fatalf("client.CallTool failed: %v", err)
	}
	return res
}

func expectTextContent(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("Incorrect number of content blocks:\n- want: 1\n-  got: %d", len(res.Content))
	}
	textContent, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Incorrect content block type:\n- want: *mcp.TextContent\n-  got: %T", res.Content[0])
	}
	return textContent.Text
}

func TestExecuteStarlark(t *testing.T) {
	client := startTestServer(t)
	params := &mcp.CallToolParams{
		Name: executeStarlarkName,
		Arguments: executeStarlarkParams{
			Program:     `print("Hello, world!")`,
			TimeoutSecs: 30,
		},
	}
	text := expectCallToolSuccess(t, client, params)
	expected := "Hello, world!\n"
	if text != expected {
		t.Fatalf("Incorrect response text:\n- want: %q\n-  got: %q", expected, text)
	}
}

func TestExecuteStarlark_Timeout(t *testing.T) {
	client := startTestServer(t)
	program := `
def main():
  for i in range(10000000): pass
main()`
	params := &mcp.CallToolParams{
		Name: executeStarlarkName,
		Arguments: executeStarlarkParams{
			Program:     program,
			TimeoutSecs: 0.1, // A very short timeout
		},
	}

	errorText := expectCallToolError(t, client, params)
	if !strings.Contains(errorText, "context deadline exceeded") {
		t.Fatalf("expected error to contain %q, but got %q", "context deadline exceeded",
			errorText)
	}
}

func TestExecuteStarlark_InvalidTimeout(t *testing.T) {
	client := startTestServer(t)
	params := &mcp.CallToolParams{
		Name: executeStarlarkName,
		Arguments: executeStarlarkParams{
			Program:     "print(1)",
			TimeoutSecs: -1.0, // Invalid timeout
		},
	}

	errorText := expectCallToolError(t, client, params)
	if !strings.Contains(errorText, "invalid timeout") {
		t.Fatalf("expected error to contain %q, but got %q", "invalid timeout",
			errorText)
	}
}

func TestExecuteStarlark_OutputBufferOverflow(t *testing.T) {
	client := startTestServer(t)

	// This will exceed the limit, since an extra newline rune is
	// added for each message.
	program := `
def main():
    large_str = "X" * 1024
    for i in range(16):
	  print(large_str)

main()
`
	params := &mcp.CallToolParams{
		Name: executeStarlarkName,
		Arguments: executeStarlarkParams{
			Program:     program,
			TimeoutSecs: 60.0,
		},
	}

	errorText := expectCallToolError(t, client, params)
	wantErrorText := "output length 16400 bytes exceeded 16384 bytes"
	if !strings.Contains(errorText, wantErrorText) {
		t.Fatalf("expected error to contain %q, but got %q", wantErrorText,
			errorText)
	}
}

func TestBuiltinsResource(t *testing.T) {
	testCases := []struct {
		name          string
		uri           string
		expectedText  string
		expectedError string
	}{
		{
			name:         "success",
			uri:          "starlark://builtins",
			expectedText: builtinsDocumentation,
		},
		{
			name:          "not_found",
			uri:           "starlark://foo",
			expectedError: "Resource not found",
		},
		{
			name:          "bad_uri",
			uri:           "://bad",
			expectedError: "Resource not found",
		},
		{
			name:          "wrong_scheme",
			uri:           "http://builtins",
			expectedError: "Resource not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := startTestServer(t)
			params := &mcp.ReadResourceParams{URI: tc.uri}
			res, err := client.ReadResource(t.Context(), params)

			if tc.expectedError != "" {
				if err == nil {
					t.Fatal("expected an error, but got none")
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("error message %q does not contain %q", err.Error(), tc.expectedError)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(res.Contents) != 1 {
				t.Fatalf("wanted len(res.Contents) = 1, got %d", len(res.Contents))
			}

			content := res.Contents[0]
			if content.Text != tc.expectedText {
				t.Fatalf("Incorrect resource content:\n- want: %q\n-  got: %q", tc.expectedText, content.Text)
			}
		})
	}
}
