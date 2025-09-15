# execute-starlark tool

Executes Starlark programs.
Starlark is a Python-like language with important restrictions and syntax differences.

## Key Differences from Python

* All top-level code must be in functions (no bare loops/conditionals)
* Operator chaining requires parentheses: use (a <= b) and (b < c) not a <= b < c  
* No f-strings or % formatting - use string concatenation with str()
* No tuple unpacking in assignments beyond simple cases
* More restrictive about operator precedence

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

* No `**` operator - use repeated multiplication: `x * x` instead of `x**2`
* No built-in functions: `sum()`, `min()`, `max()` - implement manually
* No string methods: `.rjust()`, `.strip()`, `.upper()` - implement manually
* Limited list comprehensions - avoid complex expressions like `[f(x) for x in list]`
* No `enumerate()` - use `range(len(list))` and index manually

## Common Patterns

* String building: use concatenation like s = s + "text"
* Manual sum: `total = 0; for x in numbers: total = total + x`
* String padding: implement `right_justify(text, width)` function
* Avoid complex expressions: break into multiple lines
* Use explicit str() conversion for print statements
* Put all execution logic in functions

## Built-in Functions

For documentation on available built-in functions (like `sqrt()` and `pow()` via `load("math", "sqrt", "pow")`), see the `starlark://builtins` resource.

## References

[Starlark Language Specification](https://raw.githubusercontent.com/google/starlark-go/bf296ed553ea1715656054a7f64ac6a6dd161360/doc/spec.md)