# execute-starlark tool

Executes Starlark programs.
Starlark is a Python-like language with important restrictions and syntax differences.

## Rules

* All top-level code must be in functions (no bare loops/conditionals) — call `main()` explicitly
* No while loops — use `for` with `range()`
* No recursion
* No classes, file I/O, network access, system calls, or imports (except `load()`)
* No f-strings — use `%` formatting (`"hello %s" % name`) or concatenation
* No star unpacking (`a, *b = list`)
* No `**` operator — use `x * x` or `load("math", "pow"); pow(x, 2)`
* No built-in `sum()` — use a for loop
* Operator chaining requires parentheses: `(a <= b) and (b < c)` not `a <= b < c`
* Deterministic execution only

## Example

```
def main():
  result = []
  for i in range(10):
    result.append(str(i))
  for item in result:
    print(item)

main()
```

## Available Features

* Most Python string methods (`.upper()`, `.lower()`, `.strip()`, `.replace()`, `.split()`, `.join()`, `.format()`). Missing: `.rjust()`, `.ljust()`, `.center()`
* List comprehensions: `[f(x) for x in items]`
* Math module: `load("math", "sqrt", "pow", "sin", "cos", "log", "ceil", "floor", "pi", "e")`

## Reference

[Starlark Language Specification](https://raw.githubusercontent.com/google/starlark-go/bf296ed553ea1715656054a7f64ac6a6dd161360/doc/spec.md)
