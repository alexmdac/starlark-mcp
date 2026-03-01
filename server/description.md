# execute-starlark tool

Executes Starlark programs.
Starlark is a Python-like language with some restrictions and syntax differences.

## Rules

* No classes, file I/O, network access, system calls, or imports (except `load()`)
* No f-strings — use `%` formatting (`"hello %s" % name`) or concatenation
* No star unpacking (`a, *b = list`)
* No `**` operator — use `x * x` or `load("math", "pow"); pow(x, 2)`
* Strings are not iterable — use `s.elems()`
* Operator chaining requires parentheses: `(a <= b) and (b < c)` not `a <= b < c`

## Available Features

* Most Python string methods work. Missing: `.rjust()`, `.ljust()`, `.center()`
* Math module: `load("math", "sqrt", "pow", "sin", "cos", "log", "ceil", "floor", "pi", "e")`
