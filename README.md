# Starlark MCP Server

An MCP (Model Context Protocol) server that provides Starlark code execution
capabilities to LLM clients. Starlark is a Python-like language designed for
safe, deterministic execution with built-in restrictions to prevent malicious
code. Programs can emit text output using the `print()` function, which is
captured and returned to the LLM client. This server supports the Starlark
language with custom built-in functions. For complete documentation of all
available functions, see [builtins.md](builtins.md).
