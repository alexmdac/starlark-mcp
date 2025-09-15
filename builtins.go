package main

import (
	"fmt"
	"math"
	"slices"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Built-in function inclusion criteria:
// 1. Impractical to implement well in Starlark - Requires complex algorithms or
//    substantial code
// 2. Widely useful - Applicable across many programming domains, not just
//    specialized use cases
//
// TODO:
// * sin()
// * cos()
// * PI

func checkFloat(x float64) error {
	if math.IsNaN(x) {
		return fmt.Errorf("not a number")
	}
	if math.IsInf(x, 0) {
		return fmt.Errorf("infinity")
	}
	return nil
}

func pow(
	thread *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	// TODO: also support ints and big ints.
	// TODO: support modular exponentiation.
	var x, y float64
	err := starlark.UnpackArgs(fn.Name(), args, kwargs, "x", &x, "y", &y)
	if err != nil {
		return nil, err
	}
	res := math.Pow(x, y)
	if err := checkFloat(res); err != nil {
		return nil, fmt.Errorf("pow: %v", err)
	}
	return starlark.Float(res), nil
}

func sqrt(
	thread *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	// TODO: also support ints and big ints.
	var x float64
	err := starlark.UnpackArgs(fn.Name(), args, kwargs, "x", &x)
	if err != nil {
		return nil, err
	}
	res := math.Sqrt(x)
	if err := checkFloat(res); err != nil {
		return nil, fmt.Errorf("sqrt: %v", err)
	}
	return starlark.Float(res), nil
}

func mathModule() (starlark.StringDict, error) {
	return starlark.StringDict{
		"pow":  starlark.NewBuiltin("pow", pow),
		"sqrt": starlark.NewBuiltin("sqrt", sqrt),
	}, nil
}

func loadBuiltinModule(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	switch module {
	case "math":
		return mathModule()
	default:
		return nil, fmt.Errorf("no such module: %q", module)
	}
}

func sorted(
	thread *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var iterable starlark.Iterable
	err := starlark.UnpackArgs(fn.Name(), args, kwargs, "iterable", &iterable)
	if err != nil {
		return nil, err
	}

	var elems []starlark.Value
	for elem := range starlark.Elements(iterable) {
		elems = append(elems, elem)
	}

	var sortErr error
	slices.SortFunc(elems, func(x, y starlark.Value) int {
		eql, err := starlark.Compare(syntax.EQL, x, y)
		if err != nil {
			if sortErr == nil {
				sortErr = err
			}
			return 0
		}
		if eql {
			return 0
		}
		lt, err := starlark.Compare(syntax.LT, x, y)
		if err != nil {
			if sortErr == nil {
				sortErr = err
			}
			return 0
		}
		if lt {
			return -1
		}
		return 1
	})
	if sortErr != nil {
		return nil, fmt.Errorf("sorted: %v", sortErr)
	}

	return starlark.NewList(elems), nil
}

// predeclared returns global symbols that do not need to be loaded.
func predeclared() starlark.StringDict {
	return starlark.StringDict{
		"sorted": starlark.NewBuiltin("sorted", sorted),
	}
}
