# execute-starlark tool

Executes Starlark programs.
Starlark is a Python-like language with important restrictions and syntax differences.

## Key Differences from Python

* All top-level code must be in functions (no bare loops/conditionals)
* Operator chaining requires parentheses: use (a <= b) and (b < c) not a <= b < c  
* No f-strings - use `%` formatting (`"hello %s" % name`) or string concatenation
* No star unpacking (`a, *b = list`)

## Starlark Restrictions

* No file I/O, network access, or system calls
* No imports except built-in functions
* No while loops (use for loops with range)
* No recursion allowed
* No classes or complex OOP features
* Deterministic execution only

## Example Program

```
def my_function():
  result = []
  for i in range(10):
    result.append(str(i))
  return result

def main():
  data = my_function()
  for item in data:
    print(item)

main()  # Must call explicitly
```

## Syntax Gotchas

* No `**` operator - use `load("math", "pow"); pow(x, 2)` or repeated multiplication: `x * x`
* No built-in `sum()` - implement manually
* Most Python string methods are available (`.upper()`, `.lower()`, `.strip()`, `.replace()`, `.split()`, `.join()`, `.format()`, etc). Notable missing: `.rjust()`, `.ljust()`, `.center()`
* List comprehensions work, including with function calls: `[f(x) for x in list]`

## Common Patterns

* String building: use concatenation like s = s + "text"
* Manual sum: use a for loop since `sum()` is not available
* Put all execution logic in functions (no top-level control flow)

## Built-in Functions

A `math` module is available with functions like `sqrt`, `pow`, `sin`, `cos`, `log`, `ceil`, `floor`, and constants `pi` and `e`. Use `load("math", "sqrt", "sin", "pi")` to import them. See the `starlark://builtins` resource for full documentation.

## References

[Starlark Language Specification](https://raw.githubusercontent.com/google/starlark-go/bf296ed553ea1715656054a7f64ac6a6dd161360/doc/spec.md)