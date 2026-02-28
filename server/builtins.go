package server

import (
	_ "embed"
	"fmt"

	"go.starlark.net/lib/math"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func loadBuiltinModule(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	switch module {
	case "math":
		return math.Module.Members, nil
	default:
		return nil, fmt.Errorf("no such module: %q", module)
	}
}

//go:embed prelude.star
var preludeSrc string

// prelude is computed once at init time.
var prelude starlark.StringDict

func init() {
	var err error
	prelude, err = compilePrelude()
	if err != nil {
		panic(fmt.Sprintf("prelude: %v", err))
	}
}

// compilePrelude executes prelude.star and returns its exported globals.
func compilePrelude() (starlark.StringDict, error) {
	thread := &starlark.Thread{
		Load: loadBuiltinModule,
	}
	opts := &syntax.FileOptions{
		Set:            true,
		While:          true,
		GlobalReassign: true,
		Recursion:      true,
	}
	globals, err := starlark.ExecFileOptions(opts, thread, "prelude.star", preludeSrc, nil)
	if err != nil {
		return nil, err
	}
	// Only export symbols that don't start with "_".
	exported := make(starlark.StringDict, len(globals))
	for name, val := range globals {
		if len(name) > 0 && name[0] != '_' {
			exported[name] = val
		}
	}
	return exported, nil
}

// predeclared returns global symbols that do not need to be loaded.
func predeclared() starlark.StringDict {
	return prelude
}
