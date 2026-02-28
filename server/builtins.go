package server

import (
	"fmt"

	"go.starlark.net/lib/math"
	"go.starlark.net/starlark"
)

func loadBuiltinModule(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	switch module {
	case "math":
		return math.Module.Members, nil
	default:
		return nil, fmt.Errorf("no such module: %q", module)
	}
}

// predeclared returns global symbols that do not need to be loaded.
func predeclared() starlark.StringDict {
	return starlark.StringDict{}
}
