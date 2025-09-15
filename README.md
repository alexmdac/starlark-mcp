# Starlark MCP Server

An [MCP (Model Context
Protocol)](https://modelcontextprotocol.io/docs/getting-started/intro) server
that provides Starlark code execution capabilities to LLM clients.

[Starlark](https://github.com/bazelbuild/starlark/) is a Python-like language
designed for safe, hermetic execution of untrusted code. This makes it an
attractive option for executing LLM-generated code.

Programs can emit text output using the `print()` function, which is captured
and returned to the LLM client. Output is limited to 16KB.

The server augments Starlark with some [builtin functions](builtins.md).

## Installation

You can install the server using `go install`:

```sh
go install github.com/alexmdac/starlark-mcp@latest
```

## Usage

This server is designed to be used as a plugin for an MCP-compatible agent, such
as Claude Code or Gemini.

Once installed, you can register it with your agent.

**For Claude Code:**
```sh
claude mcp add starlark-mcp starlark-mcp
```

**For Gemini:**
```sh
gemini mcp add starlark-mcp starlark-mcp
```

After registration, the agent will be able to use the `execute_starlark` tool
provided by this server.

A good first prompt to try is: *"generate a fractal using the Starlark MCP server"*.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file
for details.
