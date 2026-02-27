package main

import (
	"context"
	"strings"
	"testing"
)

// Tests that verify claims made in execute_starlark_description.md.
// If a test here fails, update the description to match reality.

func TestDescription_Works(t *testing.T) {
	tests := []struct {
		name    string
		program string
		output  string
	}{
		{"top-level expressions", `print("hello")`, "hello"},
		{"top-level assignment", `x = 1; print(x)`, "1"},
		{"operator chaining with parens", `print((1 < 2) and (2 < 3))`, "True"},
		{"percent format string", `print("hello %s" % "world")`, "hello world"},
		{"percent format int", `print("val %d" % 42)`, "val 42"},
		{"tuple unpacking", `a, b, c = 1, 2, 3; print(a, b, c)`, "1 2 3"},
		{"string upper", `print("hello".upper())`, "HELLO"},
		{"string lower", `print("HELLO".lower())`, "hello"},
		{"string strip", `print("  hi  ".strip())`, "hi"},
		{"string replace", `print("hello".replace("l", "r"))`, "herro"},
		{"string split", `print("a,b".split(","))`, `["a", "b"]`},
		{"string join", `print(",".join(["a","b"]))`, "a,b"},
		{"string format", `print("hi {}".format("there"))`, "hi there"},
		{"list comp with func call", "def double(x): return x * 2\nprint([double(x) for x in [1,2,3]])", "[2, 4, 6]"},
		{"math sqrt", `load("math", "sqrt"); print(sqrt(16))`, "4.0"},
		{"math pow", `load("math", "pow"); print(pow(2, 10))`, "1024.0"},
		{"math sin", `load("math", "sin"); print(sin(0))`, "0.0"},
		{"math cos", `load("math", "cos"); print(cos(0))`, "1.0"},
		{"math ceil", `load("math", "ceil"); print(ceil(1.5))`, "2"},
		{"math floor", `load("math", "floor"); print(floor(1.5))`, "1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executeStarlark(context.Background(), tc.program)
			if err != nil {
				t.Fatalf("expected success but got error: %v", err)
			}
			if got := strings.TrimSpace(result); got != tc.output {
				t.Fatalf("expected output %q, got %q", tc.output, got)
			}
		})
	}
}

func TestDescription_Errors(t *testing.T) {
	tests := []struct {
		name    string
		program string
		errMsg  string
	}{
		{"top-level for loop", `for i in range(1): print(i)`, "for loop not within a function"},
		{"top-level if", `if True: print("yes")`, "if statement not within a function"},
		{"operator chaining", `print(1 < 2 < 3)`, "does not associate with"},
		{"f-strings", `
def main():
    x = 42
    print(f"val {x}")
main()`, "got string literal"},
		{"star unpacking", `a, *b = [1, 2, 3]`, "got '*'"},
		{"while loop", `
def main():
    x = 0
    while x < 3:
        x += 1
main()`, "does not support while loops"},
		{"recursion", `
def fact(n):
    if n <= 1: return 1
    return n * fact(n - 1)
print(fact(5))`, "called recursively"},
		{"class", `class Foo: pass`, "got class"},
		{"power operator", `print(2 ** 10)`, "got '**'"},
		{"no sum builtin", `print(sum([1,2,3]))`, "undefined: sum"},
		{"no rjust", `print("hi".rjust(10))`, "no .rjust field or method"},
		{"no ljust", `print("hi".ljust(10))`, "no .ljust field or method"},
		{"no center", `print("hi".center(10))`, "no .center field or method"},
		{"no import", `import os`, "got import"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := executeStarlark(context.Background(), tc.program)
			if err == nil {
				t.Fatal("expected error, but program succeeded")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Fatalf("expected error containing %q, got: %v", tc.errMsg, err)
			}
		})
	}
}
