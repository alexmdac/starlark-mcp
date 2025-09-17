package main

import (
	"fmt"
	"math"

	"go.starlark.net/starlark"
)

// Built-in function inclusion criteria:
// 1. Impractical to implement well in Starlark - Requires complex algorithms or
//    substantial code
// 2. Widely useful - Applicable across many programming domains, not just
//    specialized use cases
//
// TODO:
// * sort()
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
	if x < 0 {
		return nil, fmt.Errorf("sqrt: x is negative: %f", x)
	}
	res := math.Sqrt(x)
	if err := checkFloat(res); err != nil {
		return nil, fmt.Errorf("sqrt: %v", err)
	}
	return starlark.Float(res), nil
}

func mathModule() (starlark.StringDict, error) {
	pow := starlark.NewBuiltin("pow", pow)
	sqrt := starlark.NewBuiltin("sqrt", sqrt)

	return starlark.StringDict{
		pow.Name():  pow,
		sqrt.Name(): sqrt,
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
