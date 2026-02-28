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


## Available Features

* Most Python string methods work. Missing: `.rjust()`, `.ljust()`, `.center()`
* List comprehensions: `[f(x) for x in items]`
* Math module: `load("math", "sqrt", "pow", "sin", "cos", "log", "ceil", "floor", "pi", "e")`
