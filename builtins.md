# Starlark Built-in Functions

This document describes the built-in functions available in the Starlark MCP server.

## Math Module

Mathematical functions for common operations. Import with:

```python
load("math", "sqrt", "pow")
```

### sqrt(x)

Calculate the square root of a number.

- **Parameters:** `x` (float) - The number to take the square root of
- **Returns:** `float` - The square root of x
- **Raises:** Error if x is negative

**Examples:**
```python
load("math", "sqrt")

sqrt(16.0)   # -> 4.0
sqrt(25.0)   # -> 5.0
sqrt(2.0)    # -> 1.4142135623730951
sqrt(0.0)    # -> 0.0
sqrt(-1.0)   # Error: sqrt: x is negative
```

### pow(x, y)

Raise x to the power of y (x^y).

- **Parameters:**
  - `x` (float) - The base number
  - `y` (float) - The exponent
- **Returns:** `float` - x raised to the power y
- **Raises:** Error if the result is not a valid number (NaN) or infinite

**Examples:**
```python
load("math", "pow")

pow(2.0, 3.0)    # -> 8.0 (2³)
pow(5.0, 2.0)    # -> 25.0 (5²)
pow(10.0, 0.5)   # -> 3.1622776601683795 (√10)
pow(2.0, 0.0)    # -> 1.0 (anything⁰ = 1)
pow(0.0, 2.0)    # -> 0.0 (0² = 0)
pow(-1.0, 0.5)   # Error: pow: not a number
```

## Type Requirements

All math functions currently require `float` inputs. To use with integers, convert them first:

```python
# Wrong - will cause error
sqrt(16)         # Error: got int, want float

# Correct - convert to float
sqrt(16.0)       # -> 4.0
sqrt(float(16))  # -> 4.0 (if float() conversion is available)
```

## Error Handling

Functions provide clear error messages for invalid inputs:

- **Negative square roots:** `sqrt: x is negative: -1.000000`
- **Invalid pow results:** `pow: not a number` (for operations like (-1)^0.5)
- **Type errors:** `sqrt: for parameter x: got int, want float`

## Future Enhancements

Planned improvements include:
- Support for integer inputs (automatic conversion to float)
- Modular exponentiation: `pow(base, exp, mod)`
- Additional mathematical functions as needed