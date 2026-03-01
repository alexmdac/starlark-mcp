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

The project includes an LLM eval harness that measures how effectively models
use the `execute_starlark` tool. It runs 44 test cases across 6 difficulty tiers,
executing each case multiple times to measure reliability, and produces a scored
summary with pass rates.

```sh
task eval
```

Flags can be passed after `--`:

```sh
task eval -- -runs 10 -llm anthropic:claude-sonnet-4-6
task eval -- -llm ollama:qwen3:4b -runs 1 -tier 1-2 -filter count_*
```

| Flag | Default | Description |
|------|---------|-------------|
| `-runs` | `5` | Number of independent runs per eval case |
| `-llm` | `anthropic:claude-sonnet-4-6` | `provider:model` to evaluate (see providers below) |
| `-llm-url` | per-provider default | API endpoint (overrides the provider's default URL) |
| `-filter` | | Glob pattern to select eval cases by name |
| `-tier` | | Tier filter: `N` for a single tier, `N-M` for a range |
| `-max-attempts` | `3` | Max tool-call attempts per eval case |
| `-max-iters` | `6` | Max LLM round-trips per eval case (includes nudges) |

**Providers:**

| Provider | Example | Default URL |
|----------|---------|-------------|
| `anthropic` | `anthropic:claude-sonnet-4-6` | exe.dev LLM gateway |
| `openai` | `openai:gpt-4o` | exe.dev LLM gateway |
| `ollama` | `ollama:qwen3:4b` | `http://localhost:11434` |

The `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` environment variable provides the
API key for the respective provider (optional when using the exe.dev gateway).
The `ollama` provider does not require an API key.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file
for details.
