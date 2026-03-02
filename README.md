# Starlark MCP Server

An [MCP (Model Context
Protocol)](https://modelcontextprotocol.io/docs/getting-started/intro) server
that provides Starlark code execution capabilities to LLM clients.

[Starlark](https://github.com/bazelbuild/starlark/) is a Python-like language
designed for safe, hermetic execution of untrusted code. This makes it an
attractive option for executing LLM-generated code.

Programs can emit text output using the `print()` function, which is captured
and returned to the LLM client. Output is limited to 16KB.

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

## Development

This project uses [mise](https://mise.jdx.dev/) to manage development tools.
After [installing mise](https://mise.jdx.dev/getting-started.html), set up
local development (one-time):

```sh
mise install
task setup
```

This installs [Task](https://taskfile.dev) and
[Lefthook](https://github.com/evilmartians/lefthook), and configures the
pre-push git hook.

Run checks locally:

```sh
task check
```

This runs the same checks that run automatically on `git push` via the Lefthook
pre-push hook.

See all available tasks:

```sh
task --list
```

### Evals

The project includes an LLM eval harness using
[Inspect AI](https://inspect.ai-safety-institute.org.uk/) that measures how
effectively models use the `execute_starlark` tool across 44 test cases.

Setup:

```sh
uv sync
go build -o starlark-mcp .
```

Run (the `--model` flag is required):

```sh
task eval -- --model anthropic/claude-sonnet-4-6
task eval -- --model openai/gpt-4o
task eval -- --model anthropic/claude-sonnet-4-6 --epochs 5
task eval -- --model anthropic/claude-sonnet-4-6 --limit 10
```

Set `STARLARK_MCP_BIN` to override the path to the MCP server binary
(defaults to `./starlark-mcp` in the repo root).

On exe.dev, configure the LLM gateway:

```sh
export ANTHROPIC_API_KEY=unspecified
export ANTHROPIC_BASE_URL=http://169.254.169.254/gateway/llm/anthropic
export OPENAI_API_KEY=unspecified
export OPENAI_BASE_URL=http://169.254.169.254/gateway/llm/openai/v1
```

The eval defaults to `max_tokens=4096` (set in `eval.py`), which avoids a
UTF-8 streaming issue with the exe.dev gateway proxy.

Summarize MCP tool errors from a run:

```sh
task eval:errors
```

View results with `task eval:view`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file
for details.
