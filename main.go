package main

import (
	"log"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func main() {
	program := `print("Hello, World!")`

	thread := &starlark.Thread{}
	_, err := starlark.EvalOptions(
		syntax.LegacyFileOptions(),
		thread,
		"built-in program",
		program,
		nil)
	if err != nil {
		log.Fatal(err)
	}
}
